package amtexplorer

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-xmlfmt/xmlfmt"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	ErrExplorerUseCase   = ExplorerError{Console: consoleerrors.CreateConsoleError("Unsupported Explorer Command")}
	ErrExplorerAMT       = ExplorerError{Console: consoleerrors.CreateConsoleError("AMT Error")}
	ErrExplorerNoResults = ExplorerError{Console: consoleerrors.CreateConsoleError("No results returned")}
	ErrExplorerInResult  = ExplorerError{Console: consoleerrors.CreateConsoleError("Error in result")}
	ErrNotFound          = sqldb.NotFoundError{Console: consoleerrors.CreateConsoleError("ProfilesUseCase")}
)

func (uc *UseCase) GetExplorerSupportedCalls() []string {
	var explorer AMTExplorer
	// Use reflection to get the type of the struct
	t := reflect.TypeOf(&explorer).Elem()

	methods := []string{}
	// Iterate through the methods of the struct
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		// Filter methods starting with "Get"
		if strings.HasPrefix(method.Name, "Get") {
			methods = append(methods, strings.TrimPrefix(method.Name, "Get"))
		}
	}

	return methods
}

func (uc *UseCase) ExecuteCall(ctx context.Context, guid, call, tenantID string) (*dto.Explorer, error) {
	item, err := uc.repo.GetByID(ctx, guid, tenantID)
	if err != nil {
		return &dto.Explorer{}, ErrDatabase.Wrap("ExecuteCall", "uc.repo.GetByID", err)
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, true)
	// Get the reflect.Value of the object
	objValue := reflect.ValueOf(device)

	// Get the method by name
	method := objValue.MethodByName("Get" + call)

	// Check if the method is valid
	if !method.IsValid() {
		uc.log.Warn("Method %s not found\n", call)

		return &dto.Explorer{}, ErrExplorerUseCase.Wrap("ExecuteCall", "uc.amt.Get"+call, nil)
	}

	input := make([]reflect.Value, 0)
	// invoke the method
	resultType, err := invokeMethod(input, method)
	if err != nil {
		return &dto.Explorer{}, ErrExplorerAMT.Wrap("ExecuteCall", "uc.amt.Get"+call, err)
	}

	explorer := &dto.Explorer{}
	// Iterate over the fields

	readResult(resultType, explorer)

	// Return the result
	return explorer, nil
}

func formatXML(xml string) string {
	str := xmlfmt.FormatXML(xml, "\t", "  ")

	return strings.TrimPrefix(str, "\t\r\n\t")
}

func invokeMethod(input []reflect.Value, method reflect.Value) (reflect.Value, error) {
	result := method.Call(input)

	// Ensure the result contains at least one value
	if len(result) == 0 {
		return reflect.Value{}, ErrExplorerNoResults
	}

	// Check if there is an error in the result
	if len(result) > 1 && !result[1].IsNil() {
		return reflect.Value{}, ErrExplorerInResult
	}

	// Take the first result of the method
	resultType := result[0]

	// Ensure we're working with a struct or a pointer to a struct
	if resultType.Kind() == reflect.Ptr {
		resultType = resultType.Elem()
	}

	return resultType, nil
}

func readResult(resultType reflect.Value, explorer *dto.Explorer) {
	// Iterate over the fields
	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)

		fieldName := resultType.Type().Field(i).Name

		// Check for XMLInput and XMLOutput fields
		if fieldName == "Message" {
			field = field.Elem()
			for i := 0; i < field.NumField(); i++ {
				field2 := field.Field(i)
				fieldName := field.Type().Field(i).Name

				switch fieldName {
				case "XMLInput":
					explorer.XMLInput = formatXML(field2.String())
				case "XMLOutput":
					explorer.XMLOutput = formatXML(field2.String())
				}
			}
		}
	}
}
