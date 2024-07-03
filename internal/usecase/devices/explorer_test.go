package devices_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/alarmclock"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/authorization"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/environmentdetection"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ethernetport"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/general"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/ieee8021x"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/kerberos"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/managementpresence"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/messagelog"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/mps"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publickey"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publicprivate"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/remoteaccess"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/timesynchronization"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/tls"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/userinitiatedconnection"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/wifiportconfiguration"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/bios"
	cimboot "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/card"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chassis"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chip"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/computer"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/credential"
	cimieee8021x "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/ieee8021x"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/mediaaccess"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/physical"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/processor"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/service"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/system"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/client"
	ipsalarmclock "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/ips/alarmclock"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/ips/hostbasedsetup"
	ipsieee8021x "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/ips/ieee8021x"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-amt-cloud-toolkit/console/internal/entity"
	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto"
	devices "github.com/open-amt-cloud-toolkit/console/internal/usecase/devices"
	"github.com/open-amt-cloud-toolkit/console/pkg/logger"
)

var (
	ErrExplorerGeneral = errors.New("general error")
	executeResponse    = dto.Explorer{
		XMLInput:  `<?xml version="1.0" encoding="utf-8"?><Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns="http://www.w3.org/2003/05/soap-envelope"><Header><a:Action>http://schemas.xmlsoap.org/ws/2004/09/enumeration/Pull</a:Action><a:To>/wsman</a:To><w:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</w:ResourceURI><a:MessageID>1</a:MessageID><a:ReplyTo><a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address></a:ReplyTo><w:OperationTimeout>PT60S</w:OperationTimeout></Header><Body><Pull xmlns="http://schemas.xmlsoap.org/ws/2004/09/enumeration"><EnumerationContext>4F020000-0000-0000-0000-000000000000</EnumerationContext><MaxElements>999</MaxElements><MaxCharacters>99999</MaxCharacters></Pull></Body></Envelope>`,
		XMLOutput: `<?xml version="1.0" encoding="UTF-8"?><a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope" xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:d="http://schemas.xmlsoap.org/ws/2005/02/trust" xmlns:e="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:f="http://schemas.dmtf.org/wbem/wsman/1/cimbinding.xsd" xmlns:g="http://schemas.xmlsoap.org/ws/2004/09/enumeration" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><a:Header><b:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:To><b:RelatesTo>1</b:RelatesTo><b:Action a:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/PullResponse</b:Action><b:MessageID>uuid:00000000-8086-8086-8086-0000000009F5</b:MessageID><c:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</c:ResourceURI></a:Header><a:Body><g:PullResponse><g:Items></g:Items><g:EndOfSequence></g:EndOfSequence></g:PullResponse></a:Body></a:Envelope>`,
	}
)

func initSupportedCallList(m *MockAMTExplorer) []string {
	t := reflect.TypeOf(m) // Get the type of the struct
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

func initExplorerTest(t *testing.T) (*devices.UseCase, *MockRepository, *MockAMTExplorer, dto.Explorer) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := NewMockRepository(mockCtl)
	management := NewMockManagement(mockCtl)
	amt := NewMockAMTExplorer(mockCtl)
	log := logger.New("error")
	u := devices.New(repo, management, NewMockRedirection(mockCtl), amt, log)

	return u, repo, amt, executeResponse
}

func formatXML(xml string) string {
	str := xmlfmt.FormatXML(xml, "\t", "  ")

	return strings.TrimPrefix(str, "\t\r\n\t")
}

type explorerTest struct {
	name               string
	call               string
	repoMock           func(*MockRepository)
	amtMock            func(*MockAMTExplorer)
	SupportedClassList []string
	res                any
	err                error
}

func TestGetExplorerSupportedCalls(t *testing.T) {
	t.Parallel()

	tests := []explorerTest{
		{
			name: "GetExplorerSupportedCalls",
			err:  nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc, _, amt, _ := initExplorerTest(t)

			tc.SupportedClassList = initSupportedCallList(amt)

			response := uc.GetExplorerSupportedCalls()

			require.Equal(t, tc.SupportedClassList, response)
		})
	}
}

func TestExecuteCall(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []explorerTest{
		{
			name: "ExecuteCall GetById fails",
			call: "ById",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(nil, ErrExplorerGeneral)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(context.Background(), false, true).
					Return()
			},
			res: &dto.Explorer{},
			err: devices.ErrDatabase,
		},
		{
			name: "ExecuteCall Unsupported Explorer Command",
			call: "NotSupportedCommand",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMT8021xCredentialContextSuccess",
			call: "AMT8021xCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				amt.EXPECT().
					GetAMT8021xCredentialContext().
					Return(ieee8021x.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMT8021xCredentialContextError",
			call: "AMT8021xCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				amt.EXPECT().
					GetAMT8021xCredentialContext().
					Return(ieee8021x.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMT8021xProfileSuccess",
			call: "AMT8021xProfile",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				amt.EXPECT().
					GetAMT8021xProfile().
					Return(ieee8021x.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMT8021xProfileError",
			call: "AMT8021xProfile",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				amt.EXPECT().
					GetAMT8021xProfile().
					Return(ieee8021x.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTAlarmClockServiceSuccess",
			call: "AMTAlarmClockService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				amt.EXPECT().
					GetAMTAlarmClockService().
					Return(alarmclock.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: dto.Explorer{
				XMLInput:  `<?xml version="1.0" encoding="utf-8"?><Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns="http://www.w3.org/2003/05/soap-envelope"><Header><a:Action>http://schemas.xmlsoap.org/ws/2004/09/enumeration/Pull</a:Action><a:To>/wsman</a:To><w:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</w:ResourceURI><a:MessageID>1</a:MessageID><a:ReplyTo><a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address></a:ReplyTo><w:OperationTimeout>PT60S</w:OperationTimeout></Header><Body><Pull xmlns="http://schemas.xmlsoap.org/ws/2004/09/enumeration"><EnumerationContext>4F020000-0000-0000-0000-000000000000</EnumerationContext><MaxElements>999</MaxElements><MaxCharacters>99999</MaxCharacters></Pull></Body></Envelope>`,
				XMLOutput: `<?xml version="1.0" encoding="UTF-8"?><a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope" xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:d="http://schemas.xmlsoap.org/ws/2005/02/trust" xmlns:e="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:f="http://schemas.dmtf.org/wbem/wsman/1/cimbinding.xsd" xmlns:g="http://schemas.xmlsoap.org/ws/2004/09/enumeration" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><a:Header><b:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:To><b:RelatesTo>1</b:RelatesTo><b:Action a:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/PullResponse</b:Action><b:MessageID>uuid:00000000-8086-8086-8086-0000000009F5</b:MessageID><c:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</c:ResourceURI></a:Header><a:Body><g:PullResponse><g:Items></g:Items><g:EndOfSequence></g:EndOfSequence></g:PullResponse></a:Body></a:Envelope>`,
			},
			err: nil,
		},
		{
			name: "getAMTAlarmClockServiceError",
			call: "AMTAlarmClockService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(amt *MockAMTExplorer) {
				amt.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				amt.EXPECT().
					GetAMTAlarmClockService().
					Return(alarmclock.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTAuditLogSuccess",
			call: "AMTAuditLog",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTAuditLog().
					Return(auditlog.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: dto.Explorer{
				XMLInput:  `<?xml version="1.0" encoding="utf-8"?><Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns="http://www.w3.org/2003/05/soap-envelope"><Header><a:Action>http://schemas.xmlsoap.org/ws/2004/09/enumeration/Pull</a:Action><a:To>/wsman</a:To><w:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</w:ResourceURI><a:MessageID>1</a:MessageID><a:ReplyTo><a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address></a:ReplyTo><w:OperationTimeout>PT60S</w:OperationTimeout></Header><Body><Pull xmlns="http://schemas.xmlsoap.org/ws/2004/09/enumeration"><EnumerationContext>4F020000-0000-0000-0000-000000000000</EnumerationContext><MaxElements>999</MaxElements><MaxCharacters>99999</MaxCharacters></Pull></Body></Envelope>`,
				XMLOutput: `<?xml version="1.0" encoding="UTF-8"?><a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope" xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:d="http://schemas.xmlsoap.org/ws/2005/02/trust" xmlns:e="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:f="http://schemas.dmtf.org/wbem/wsman/1/cimbinding.xsd" xmlns:g="http://schemas.xmlsoap.org/ws/2004/09/enumeration" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><a:Header><b:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:To><b:RelatesTo>1</b:RelatesTo><b:Action a:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/PullResponse</b:Action><b:MessageID>uuid:00000000-8086-8086-8086-0000000009F5</b:MessageID><c:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</c:ResourceURI></a:Header><a:Body><g:PullResponse><g:Items></g:Items><g:EndOfSequence></g:EndOfSequence></g:PullResponse></a:Body></a:Envelope>`,
			},
			err: nil,
		},
		{
			name: "getAMTAuditLogError",
			call: "AMTAuditLog",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTAuditLog().
					Return(auditlog.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTAuthorizationServiceSuccess",
			call: "AMTAuthorizationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTAuthorizationService().
					Return(authorization.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: dto.Explorer{
				XMLInput:  `<?xml version="1.0" encoding="utf-8"?><Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns="http://www.w3.org/2003/05/soap-envelope"><Header><a:Action>http://schemas.xmlsoap.org/ws/2004/09/enumeration/Pull</a:Action><a:To>/wsman</a:To><w:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</w:ResourceURI><a:MessageID>1</a:MessageID><a:ReplyTo><a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address></a:ReplyTo><w:OperationTimeout>PT60S</w:OperationTimeout></Header><Body><Pull xmlns="http://schemas.xmlsoap.org/ws/2004/09/enumeration"><EnumerationContext>4F020000-0000-0000-0000-000000000000</EnumerationContext><MaxElements>999</MaxElements><MaxCharacters>99999</MaxCharacters></Pull></Body></Envelope>`,
				XMLOutput: `<?xml version="1.0" encoding="UTF-8"?><a:Envelope xmlns:a="http://www.w3.org/2003/05/soap-envelope" xmlns:b="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:c="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:d="http://schemas.xmlsoap.org/ws/2005/02/trust" xmlns:e="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:f="http://schemas.dmtf.org/wbem/wsman/1/cimbinding.xsd" xmlns:g="http://schemas.xmlsoap.org/ws/2004/09/enumeration" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><a:Header><b:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:To><b:RelatesTo>1</b:RelatesTo><b:Action a:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/PullResponse</b:Action><b:MessageID>uuid:00000000-8086-8086-8086-0000000009F5</b:MessageID><c:ResourceURI>http://intel.com/wbem/wscim/1/amt-schema/1/AMT_8021xCredentialContext</c:ResourceURI></a:Header><a:Body><g:PullResponse><g:Items></g:Items><g:EndOfSequence></g:EndOfSequence></g:PullResponse></a:Body></a:Envelope>`,
			},
			err: nil,
		},
		{
			name: "getAMTAuthorizationServiceError",
			call: "AMTAuthorizationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTAuthorizationService().
					Return(authorization.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTBootCapabilitiesSuccess",
			call: "AMTBootCapabilities",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTBootCapabilities().
					Return(boot.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTBootCapabilitiesError",
			call: "AMTBootCapabilities",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTBootCapabilities().
					Return(boot.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTBootSettingDataSuccess",
			call: "AMTBootSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTBootSettingData().
					Return(boot.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTBootSettingDataError",
			call: "AMTBootSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTBootSettingData().
					Return(boot.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTEnvironmentDetectionSettingDataSuccess",
			call: "AMTEnvironmentDetectionSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTEnvironmentDetectionSettingData().
					Return(environmentdetection.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTEnvironmentDetectionSettingDataError",
			call: "AMTEnvironmentDetectionSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTEnvironmentDetectionSettingData().
					Return(environmentdetection.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTEthernetPortSettingsSuccess",
			call: "AMTEthernetPortSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTEthernetPortSettings().
					Return(ethernetport.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTEthernetPortSettingsError",
			call: "AMTEthernetPortSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTEthernetPortSettings().
					Return(ethernetport.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTGeneralSettingsSuccess",
			call: "AMTGeneralSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTGeneralSettings().
					Return(general.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTGeneralSettingsError",
			call: "AMTGeneralSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTGeneralSettings().
					Return(general.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTKerberosSettingDataSuccess",
			call: "AMTKerberosSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTKerberosSettingData().
					Return(kerberos.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTKerberosSettingDataError",
			call: "AMTKerberosSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTKerberosSettingData().
					Return(kerberos.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTManagementPresenceRemoteSAPSuccess",
			call: "AMTManagementPresenceRemoteSAP",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTManagementPresenceRemoteSAP().
					Return(managementpresence.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTManagementPresenceRemoteSAPError",
			call: "AMTManagementPresenceRemoteSAP",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTManagementPresenceRemoteSAP().
					Return(managementpresence.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTMessageLogSuccess",
			call: "AMTMessageLog",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTMessageLog().
					Return(messagelog.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTMessageLogError",
			call: "AMTMessageLog",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTMessageLog().
					Return(messagelog.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTMPSUsernamePasswordSuccess",
			call: "AMTMPSUsernamePassword",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTMPSUsernamePassword().
					Return(mps.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTMPSUsernamePasswordError",
			call: "AMTMPSUsernamePassword",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTMPSUsernamePassword().
					Return(mps.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTPublicKeyCertificateSuccess",
			call: "AMTPublicKeyCertificate",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTPublicKeyCertificate().
					Return(publickey.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTPublicKeyCertificateError",
			call: "AMTPublicKeyCertificate",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTPublicKeyCertificate().
					Return(publickey.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTPublicKeyManagementServiceSuccess",
			call: "AMTPublicKeyManagementService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTPublicKeyManagementService().
					Return(publickey.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTPublicKeyManagementServiceError",
			call: "AMTPublicKeyManagementService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTPublicKeyManagementService().
					Return(publickey.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTPublicPrivateKeyPairSuccess",
			call: "AMTPublicPrivateKeyPair",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTPublicPrivateKeyPair().
					Return(publicprivate.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTPublicPrivateKeyPairError",
			call: "AMTPublicPrivateKeyPair",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTPublicPrivateKeyPair().
					Return(publicprivate.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTRedirectionServiceSuccess",
			call: "AMTRedirectionService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTRedirectionServiceError",
			call: "AMTRedirectionService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTRemoteAccessPolicyAppliesToMPSSuccess",
			call: "AMTRemoteAccessPolicyAppliesToMPS",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRemoteAccessPolicyAppliesToMPS().
					Return(remoteaccess.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTRemoteAccessPolicyAppliesToMPSError",
			call: "AMTRemoteAccessPolicyAppliesToMPS",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRemoteAccessPolicyAppliesToMPS().
					Return(remoteaccess.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTRemoteAccessPolicyRuleSuccess",
			call: "AMTRemoteAccessPolicyRule",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRemoteAccessPolicyRule().
					Return(remoteaccess.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTRemoteAccessPolicyRuleError",
			call: "AMTRemoteAccessPolicyRule",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRemoteAccessPolicyRule().
					Return(remoteaccess.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTRemoteAccessServiceSuccess",
			call: "AMTRemoteAccessService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRemoteAccessService().
					Return(remoteaccess.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTRemoteAccessServiceError",
			call: "AMTRemoteAccessService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTRemoteAccessService().
					Return(remoteaccess.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTSetupAndConfigurationServiceSuccess",
			call: "AMTSetupAndConfigurationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTSetupAndConfigurationService().
					Return(setupandconfiguration.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTSetupAndConfigurationServiceError",
			call: "AMTSetupAndConfigurationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTSetupAndConfigurationService().
					Return(setupandconfiguration.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTTimeSynchronizationServiceSuccess",
			call: "AMTTimeSynchronizationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTimeSynchronizationService().
					Return(timesynchronization.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTTimeSynchronizationServiceError",
			call: "AMTTimeSynchronizationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTimeSynchronizationService().
					Return(timesynchronization.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTTLSCredentialContextSuccess",
			call: "AMTTLSCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTLSCredentialContext().
					Return(tls.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTTLSCredentialContextError",
			call: "AMTTLSCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTLSCredentialContext().
					Return(tls.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTTLSProtocolEndpointCollectionSuccess",
			call: "AMTTLSProtocolEndpointCollection",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTLSProtocolEndpointCollection().
					Return(tls.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTTLSProtocolEndpointCollectionError",
			call: "AMTTLSProtocolEndpointCollection",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTLSProtocolEndpointCollection().
					Return(tls.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTTLSSettingDataSuccess",
			call: "AMTTLSSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTLSSettingData().
					Return(tls.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTAMTTLSSettingDataError",
			call: "AMTAMTTLSSettingData",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTTLSSettingData().
					Return(tls.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTUserInitiatedConnectionServiceSuccess",
			call: "AMTUserInitiatedConnectionService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTUserInitiatedConnectionService().
					Return(userinitiatedconnection.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTUserInitiatedConnectionServiceError",
			call: "AMTUserInitiatedConnectionService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTUserInitiatedConnectionService().
					Return(userinitiatedconnection.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getAMTWiFiPortConifgurationServiceSuccess",
			call: "AMTWiFiPortConifgurationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTWiFiPortConifgurationService().
					Return(wifiportconfiguration.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getAMTWiFiPortConifgurationServiceError",
			call: "AMTWiFiPortConifgurationService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetAMTWiFiPortConifgurationService().
					Return(wifiportconfiguration.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMBIOSElementSuccess",
			call: "CIMBIOSElement",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBIOSElement().
					Return(bios.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMBIOSElementError",
			call: "CIMBIOSElement",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBIOSElement().
					Return(bios.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMBootConfigSettingSuccess",
			call: "CIMBootConfigSetting",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBootConfigSetting().
					Return(cimboot.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMBootConfigSettingError",
			call: "CIMBootConfigSetting",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBootConfigSetting().
					Return(cimboot.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMBootServiceSuccess",
			call: "CIMBootService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBootService().
					Return(cimboot.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMBootServiceError",
			call: "CIMBootService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBootService().
					Return(cimboot.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMBootSourceSettingSuccess",
			call: "CIMBootSourceSetting",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimboot.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMBootSourceSettingError",
			call: "CIMBootSourceSetting",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimboot.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMCardSuccess",
			call: "CIMCard",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMCard().
					Return(card.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMCardError",
			call: "CIMCard",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMCard().
					Return(card.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMChassisSuccess",
			call: "CIMChassis",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMChassis().
					Return(chassis.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMChassisError",
			call: "CIMChassis",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMChassis().
					Return(chassis.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMChipSuccess",
			call: "CIMChip",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMChip().
					Return(chip.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMChipError",
			call: "CIMChip",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMChip().
					Return(chip.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMComputerSystemPackageSuccess",
			call: "CIMComputerSystemPackage",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMComputerSystemPackage().
					Return(computer.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMComputerSystemPackageError",
			call: "CIMComputerSystemPackage",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMComputerSystemPackage().
					Return(computer.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMConcreteDependencySuccess",
			call: "CIMConcreteDependency",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMConcreteDependency().
					Return(concrete.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMConcreteDependencyError",
			call: "CIMConcreteDependency",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMConcreteDependency().
					Return(concrete.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMCredentialContextSuccess",
			call: "CIMCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMCredentialContext().
					Return(credential.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMCredentialContextError",
			call: "CIMCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMCredentialContext().
					Return(credential.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMIEEE8021xSettingsSuccess",
			call: "CIMIEEE8021xSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMIEEE8021xSettings().
					Return(cimieee8021x.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMIEEE8021xSettingsError",
			call: "CIMIEEE8021xSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMIEEE8021xSettings().
					Return(cimieee8021x.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMKVMRedirectionSAPSuccess",
			call: "CIMKVMRedirectionSAP",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMKVMRedirectionSAP().
					Return(kvm.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMKVMRedirectionSAPError",
			call: "CIMKVMRedirectionSAP",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMKVMRedirectionSAP().
					Return(kvm.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMMediaAccessDeviceSuccess",
			call: "CIMMediaAccessDevice",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMMediaAccessDevice().
					Return(mediaaccess.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMMediaAccessDeviceError",
			call: "CIMMediaAccessDevice",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMMediaAccessDevice().
					Return(mediaaccess.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMPhysicalMemorySuccess",
			call: "CIMPhysicalMemory",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMPhysicalMemory().
					Return(physical.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMPhysicalMemoryError",
			call: "CIMPhysicalMemory",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMPhysicalMemory().
					Return(physical.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMPhysicalPackageSuccess",
			call: "CIMPhysicalPackage",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMPhysicalPackage().
					Return(physical.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMPhysicalPackageError",
			call: "CIMPhysicalPackage",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMPhysicalPackage().
					Return(physical.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMPowerManagementServiceSuccess",
			call: "CIMPowerManagementService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMPowerManagementService().
					Return(power.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMPowerManagementServiceError",
			call: "CIMPowerManagementService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMPowerManagementService().
					Return(power.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMProcessorSuccess",
			call: "CIMProcessor",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMProcessor().
					Return(processor.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMProcessorError",
			call: "CIMProcessor",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMProcessor().
					Return(processor.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMServiceAvailableToElementSuccess",
			call: "CIMServiceAvailableToElement",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMServiceAvailableToElement().
					Return(service.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMServiceAvailableToElementError",
			call: "CIMServiceAvailableToElement",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMServiceAvailableToElement().
					Return(service.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMSoftwareIdentitySuccess",
			call: "CIMSoftwareIdentity",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMSoftwareIdentity().
					Return(software.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMSoftwareIdentityError",
			call: "CIMSoftwareIdentity",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMSoftwareIdentity().
					Return(software.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMSystemPackagingSuccess",
			call: "CIMSystemPackaging",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMSystemPackaging().
					Return(system.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMSystemPackagingError",
			call: "CIMSystemPackaging",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMSystemPackaging().
					Return(system.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMWiFiEndpointSettingsSuccess",
			call: "CIMWiFiEndpointSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMWiFiEndpointSettings().
					Return(wifi.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMWiFiEndpointSettingsError",
			call: "CIMWiFiEndpointSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMWiFiEndpointSettings().
					Return(wifi.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getCIMWiFiPortSuccess",
			call: "CIMWiFiPort",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMWiFiPort().
					Return(wifi.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getCIMWiFiPortError",
			call: "CIMWiFiPort",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetCIMWiFiPort().
					Return(wifi.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getIPS8021xCredentialContextSuccess",
			call: "IPS8021xCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPS8021xCredentialContext().
					Return(ipsieee8021x.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getIPS8021xCredentialContextError",
			call: "IPS8021xCredentialContext",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPS8021xCredentialContext().
					Return(ipsieee8021x.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getIPSAlarmClockOccurrenceSuccess",
			call: "IPSAlarmClockOccurrence",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSAlarmClockOccurrence().
					Return(ipsalarmclock.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getIPSAlarmClockOccurrenceError",
			call: "IPSAlarmClockOccurrence",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSAlarmClockOccurrence().
					Return(ipsalarmclock.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getIPSHostBasedSetupServiceSuccess",
			call: "IPSHostBasedSetupService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSHostBasedSetupService().
					Return(hostbasedsetup.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getIPSHostBasedSetupServiceError",
			call: "IPSHostBasedSetupService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSHostBasedSetupService().
					Return(hostbasedsetup.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getIPSIEEE8021xSettingsSuccess",
			call: "IPSIEEE8021xSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSIEEE8021xSettings().
					Return(ipsieee8021x.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getIPSIEEE8021xSettingsError",
			call: "IPSIEEE8021xSettings",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSIEEE8021xSettings().
					Return(ipsieee8021x.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
		{
			name: "getIPSOptInServiceSuccess",
			call: "IPSOptInService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{Message: &client.Message{XMLInput: executeResponse.XMLInput, XMLOutput: executeResponse.XMLOutput}}, nil)
			},
			res: executeResponse,
			err: nil,
		},
		{
			name: "getIPSOptInServiceError",
			call: "IPSOptInService",
			repoMock: func(repo *MockRepository) {
				repo.EXPECT().
					GetByID(gomock.Any(), device.GUID, device.TenantID).
					Return(device, nil)
			},
			amtMock: func(man *MockAMTExplorer) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return()
				man.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{}, ErrExplorerGeneral)
			},
			res: &dto.Explorer{},
			err: devices.ErrExplorerUseCase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc, repo, amt, executeResponse := initExplorerTest(t)

			tc.amtMock(amt)
			tc.repoMock(repo)

			res, err := uc.ExecuteCall(context.Background(), device.GUID, tc.call, device.TenantID)
			if res.XMLInput != "" {
				formattedXMLInput := formatXML(executeResponse.XMLInput)
				formattedXMLOutput := formatXML(executeResponse.XMLOutput)

				tc.res = &dto.Explorer{
					XMLInput:  formattedXMLInput,
					XMLOutput: formattedXMLOutput,
				}
			}

			require.IsType(t, tc.err, err)
			require.Equal(t, tc.res, res)
		})
	}
}
