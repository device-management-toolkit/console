package openapi

import (
	"encoding/json"
	"os"

	httpController "github.com/device-management-toolkit/console/internal/controller/http"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Generator handles OpenAPI specification generation
type Generator struct {
	usecases usecase.Usecases
	logger   logger.Interface
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(usecases usecase.Usecases, logger logger.Interface) *Generator {
	return &Generator{
		usecases: usecases,
		logger:   logger,
	}
}

// GenerateSpec generates OpenAPI 3.1.0 specification with compliance fixes
func (g *Generator) GenerateSpec() ([]byte, int, int, error) {
	adapter := httpController.NewFuegoAdapter(g.usecases, g.logger)
	adapter.RegisterRoutes()

	spec, err := adapter.GetOpenAPISpec()
	if err != nil {
		return nil, 0, 0, err
	}

	var specJSON map[string]interface{}
	if err := json.Unmarshal(spec, &specJSON); err != nil {
		return nil, 0, 0, err
	}

	fixedCount := FixCompliance(specJSON)
	endpointCount := CountEndpoints(specJSON)

	finalSpec, err := json.MarshalIndent(specJSON, "", "  ")
	if err != nil {
		return nil, 0, 0, err
	}

	return finalSpec, endpointCount, fixedCount, nil
}

// SaveSpec saves the OpenAPI specification to a file
func (g *Generator) SaveSpec(spec []byte, filePath string) error {
	return os.WriteFile(filePath, spec, 0644)
}

// FixCompliance fixes OpenAPI 3.1.0 compliance issues
func FixCompliance(data interface{}) int {
	fixedCount := 0

	switch v := data.(type) {
	case map[string]interface{}:
		// Remove empty keys
		if _, exists := v[""]; exists {
			delete(v, "")
			fixedCount++
		}

		// Fix invalid schema references
		if ref, ok := v["$ref"].(string); ok && ref == "#/components/schemas/" {
			delete(v, "$ref")
			v["type"] = "object"
			v["description"] = "Generic response object"
			fixedCount++
		}

		// Convert nullable to union type
		if nullable, ok := v["nullable"].(bool); ok && nullable {
			if schemaType, typeOk := v["type"].(string); typeOk {
				delete(v, "nullable")
				v["type"] = []interface{}{schemaType, "null"}
				fixedCount++
			}
		}

		// Convert example to examples array
		if example, ok := v["example"]; ok {
			delete(v, "example")
			v["examples"] = []interface{}{example}
			fixedCount++
		}

		// Convert examples object to array format
		if examples, ok := v["examples"].(map[string]interface{}); ok {
			if defaultExample, hasDefault := examples["default"]; hasDefault {
				if defaultObj, isObj := defaultExample.(map[string]interface{}); isObj {
					if value, hasValue := defaultObj["value"]; hasValue {
						v["examples"] = []interface{}{value}
						fixedCount++
					}
				}
			}
		}

		// Recursively fix nested objects
		for _, value := range v {
			fixedCount += FixCompliance(value)
		}

	case []interface{}:
		// Recursively fix array elements
		for _, item := range v {
			fixedCount += FixCompliance(item)
		}
	}

	return fixedCount
}

// CountEndpoints counts API endpoints in the specification
func CountEndpoints(spec map[string]interface{}) int {
	count := 0
	if paths, ok := spec["paths"].(map[string]interface{}); ok {
		for _, pathMethods := range paths {
			if methods, ok := pathMethods.(map[string]interface{}); ok {
				for method := range methods {
					if method != "description" && method != "summary" {
						count++
					}
				}
			}
		}
	}
	return count
}
