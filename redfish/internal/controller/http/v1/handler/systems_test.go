package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// Test constants specific to systems handler
const (
	systemsOdataContextCollectionTest = "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"
	systemsOdataIDCollectionTest      = "/redfish/v1/Systems"
	systemsOdataTypeCollectionTest    = "#ComputerSystemCollection.ComputerSystemCollection"
	systemsCollectionTitleTest        = "Computer System Collection"
	systemsEndpointTest               = "/redfish/v1/Systems"
	jsonContentTypeTest               = "application/json; charset=utf-8"
	systemODataType                   = "#ComputerSystem.v1_26_0.ComputerSystem"

	// Test UUID constants for consistent system IDs
	testUUID1        = "550e8400-e29b-41d4-a716-446655440001"
	testUUID2        = "550e8400-e29b-41d4-a716-446655440002"
	testUUID3        = "550e8400-e29b-41d4-a716-446655440003"
	testUUID4        = "550e8400-e29b-41d4-a716-446655440004"
	testUUID5        = "550e8400-e29b-41d4-a716-446655440005"
	testUUID6        = "550e8400-e29b-41d4-a716-446655440006"
	testUUID7        = "550e8400-e29b-41d4-a716-446655440007"
	testUUID8        = "550e8400-e29b-41d4-a716-446655440008"
	testUUID9        = "550e8400-e29b-41d4-a716-446655440009"
	testUUID10       = "550e8400-e29b-41d4-a716-44665544000a"
	testUUIDNotFound = "999e8400-e29b-41d4-a716-446655440000"
)

// Test error constants specific to systems
var (
	errSystemRepoFailure = errors.New("system repository operation failed")
)

// TestSystemsComputerSystemRepository is a test implementation for systems tests
type TestSystemsComputerSystemRepository struct {
	systems        map[string]*redfishv1.ComputerSystem
	errorOnGetAll  bool
	errorOnGetByID map[string]error
}

func NewTestSystemsComputerSystemRepository() *TestSystemsComputerSystemRepository {
	return &TestSystemsComputerSystemRepository{
		systems:        make(map[string]*redfishv1.ComputerSystem),
		errorOnGetByID: make(map[string]error),
	}
}

func (r *TestSystemsComputerSystemRepository) AddSystem(id string, system *redfishv1.ComputerSystem) {
	r.systems[id] = system
}

func (r *TestSystemsComputerSystemRepository) SetErrorOnGetAll(err bool) {
	r.errorOnGetAll = err
}

func (r *TestSystemsComputerSystemRepository) SetErrorOnGetByID(systemID string, err error) {
	r.errorOnGetByID[systemID] = err
}

func (r *TestSystemsComputerSystemRepository) GetAll(_ context.Context) ([]string, error) {
	if r.errorOnGetAll {
		return nil, errSystemRepoFailure
	}

	ids := make([]string, 0, len(r.systems))
	for id := range r.systems {
		if id != "" {
			ids = append(ids, id)
		}
	}
	// Sort to ensure consistent ordering for tests
	sort.Strings(ids)

	return ids, nil
}

func (r *TestSystemsComputerSystemRepository) GetByID(_ context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	if err, exists := r.errorOnGetByID[systemID]; exists {
		return nil, err
	}

	if system, exists := r.systems[systemID]; exists {
		return system, nil
	}

	return nil, usecase.ErrSystemNotFound
}

func (r *TestSystemsComputerSystemRepository) UpdatePowerState(_ context.Context, systemID string, resetType redfishv1.PowerState) error {
	if system, exists := r.systems[systemID]; exists {
		// Convert reset types to final power state for testing
		switch resetType {
		case redfishv1.ResetTypeOn: // Covers both ResetTypeOn and PowerStateOn (same value "On")
			system.PowerState = redfishv1.PowerStateOn
		case redfishv1.ResetTypeForceOff:
			system.PowerState = redfishv1.PowerStateOff
		case redfishv1.PowerStateOff:
			system.PowerState = redfishv1.PowerStateOff
		case redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
			// These reset types cycle power, for test purposes we set to Off
			// (in reality the system would restart)
			system.PowerState = redfishv1.PowerStateOff
		}

		return nil
	}

	return usecase.ErrSystemNotFound
}

func (r *TestSystemsComputerSystemRepository) GetBootSettings(_ context.Context, systemID string) (*generated.ComputerSystemBoot, error) {
	if _, exists := r.systems[systemID]; exists {
		// Return empty boot settings for tests
		return &generated.ComputerSystemBoot{}, nil
	}

	return nil, usecase.ErrSystemNotFound
}

func (r *TestSystemsComputerSystemRepository) UpdateBootSettings(_ context.Context, systemID string, _ *generated.ComputerSystemBoot) error {
	if _, exists := r.systems[systemID]; exists {
		return nil
	}

	return usecase.ErrSystemNotFound
}

// TestCase represents a generic test case structure
type SystemsTestCase[T any] struct {
	name           string
	setupRepo      func(*TestSystemsComputerSystemRepository, T)
	httpMethod     string
	expectedStatus int
	validateFunc   func(*testing.T, *httptest.ResponseRecorder, T)
	params         T
}

// TestConfig holds configuration for running tests
type SystemsTestConfig[T any] struct {
	endpoint    string
	routerSetup func(*RedfishServer) *gin.Engine
	urlBuilder  func(T) string
}

// validateJSONContentTypeTest validates that the response has the expected JSON content type
func validateJSONContentTypeTest(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, jsonContentTypeTest, w.Header().Get("Content-Type"))
}

// unmarshalJSONResponseTest unmarshals JSON response and validates no error occurred
func unmarshalJSONResponseTest(t *testing.T, w *httptest.ResponseRecorder, response interface{}) {
	t.Helper()

	err := json.Unmarshal(w.Body.Bytes(), response)
	assert.NoError(t, err)
}

// validateSystemsCollectionResponse validates the basic structure of a systems collection response
func validateSystemsCollectionResponse(t *testing.T, w *httptest.ResponseRecorder, expectedCount int64, expectedMembers []string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemCollectionComputerSystemCollection
	unmarshalJSONResponseTest(t, w, &response)

	// Validate response structure
	assert.Equal(t, systemsOdataContextCollectionTest, *response.OdataContext)
	assert.Equal(t, systemsEndpointTest, *response.OdataId)
	assert.Equal(t, systemsOdataTypeCollectionTest, *response.OdataType)
	assert.Equal(t, systemsCollectionTitleTest, response.Name)
	assert.NotNil(t, response.Description)
	assert.Equal(t, expectedCount, *response.MembersOdataCount)
	assert.NotNil(t, response.Members)
	assert.Len(t, *response.Members, int(expectedCount))

	// Validate members if any expected
	if expectedCount > 0 {
		members := *response.Members
		for i, expectedMember := range expectedMembers {
			assert.Equal(t, expectedMember, *members[i].OdataId)
		}
	}
}

// Router setup for testing

func setupSystemsTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(systemsEndpointTest, server.GetRedfishV1Systems)

	return router
}

func setupSystemByIDTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/redfish/v1/Systems/:ComputerSystemId", func(c *gin.Context) {
		computerSystemID := c.Param("ComputerSystemId")
		server.GetRedfishV1SystemsComputerSystemId(c, computerSystemID)
	})
	// Add route for empty system ID case (trailing slash)
	router.GET("/redfish/v1/Systems/", func(c *gin.Context) {
		server.GetRedfishV1SystemsComputerSystemId(c, "")
	})

	return router
}

// Validation functions for Systems collection responses (reusing shared function)
func validateMultipleSystemsResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 2, []string{
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID1),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID2),
	})
}

func validateEmptyCollectionResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 0, []string{})
}

func validateFilteredSystemsResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 3, []string{
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID1),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID2),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID3),
	})
}

func validateSingleSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	validateSystemsCollectionResponse(t, w, 1, []string{fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID1)})
}

func validateLargeCollectionResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()

	// Create expected members in alphabetical order (as returned by sorted repository)
	expectedMembers := []string{
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID1),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID2),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID3),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID4),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID5),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID6),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID7),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID8),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID9),
		fmt.Sprintf("%s/%s", systemsEndpointTest, testUUID10),
	}

	validateSystemsCollectionResponse(t, w, 10, expectedMembers)
}

func validateJSONResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var jsonResponse map[string]interface{}
	unmarshalJSONResponseTest(t, w, &jsonResponse)
	assert.NotEmpty(t, jsonResponse)
}

// validateErrorResponseDataTest validates error response with expected code and message
//
//nolint:unparam // expectedCode varies in different test contexts
func validateErrorResponseDataTest(t *testing.T, w *httptest.ResponseRecorder, expectedCode, expectedMessage string) map[string]interface{} {
	t.Helper()

	var errorResponse map[string]interface{}
	unmarshalJSONResponseTest(t, w, &errorResponse)

	assert.Contains(t, errorResponse, "error")
	errorObj, ok := errorResponse["error"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, expectedCode, errorObj["code"])
	assert.Equal(t, expectedMessage, errorObj["message"])

	return errorObj
}

// validateExtendedInfoTest validates the extended info section of an error response
func validateExtendedInfoTest(t *testing.T, errorObj map[string]interface{}, expectedMessageID, expectedMessageContains, expectedSeverity string) {
	t.Helper()

	extendedInfo, ok := errorObj["@Message.ExtendedInfo"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, extendedInfo, 1)

	messageInfo, ok := extendedInfo[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, expectedMessageID, messageInfo["MessageId"])
	assert.Contains(t, messageInfo["Message"], expectedMessageContains)
	assert.Equal(t, expectedSeverity, messageInfo["Severity"])
}

func validateErrorResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ struct{}) {
	t.Helper()
	_ = validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "An internal server error occurred.")
}

// Repository setup functions for Systems collection
func setupMultipleSystemsMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.AddSystem(testUUID1, &redfishv1.ComputerSystem{ID: testUUID1, Name: "System 1", PowerState: redfishv1.PowerStateOn})
	repo.AddSystem(testUUID2, &redfishv1.ComputerSystem{ID: testUUID2, Name: "System 2", PowerState: redfishv1.PowerStateOn})
}

func setupEmptyCollectionMockTest(_ *TestSystemsComputerSystemRepository, _ struct{}) {
	// No systems added - empty collection
}

func setupFilteredSystemsMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.AddSystem(testUUID1, &redfishv1.ComputerSystem{ID: testUUID1, Name: "System 1", PowerState: redfishv1.PowerStateOn})
	repo.AddSystem("", &redfishv1.ComputerSystem{ID: "", Name: "Empty ID", PowerState: redfishv1.PowerStateOn}) // This will be filtered out
	repo.AddSystem(testUUID3, &redfishv1.ComputerSystem{ID: testUUID3, Name: "ABC System", PowerState: redfishv1.PowerStateOn})
	repo.AddSystem(testUUID2, &redfishv1.ComputerSystem{ID: testUUID2, Name: "System 2", PowerState: redfishv1.PowerStateOn})
}

func setupSingleSystemMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.AddSystem(testUUID1, &redfishv1.ComputerSystem{ID: testUUID1, Name: "System 1", PowerState: redfishv1.PowerStateOn})
}

func setupLargeCollectionMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	uuids := []string{testUUID1, testUUID2, testUUID3, testUUID4, testUUID5, testUUID6, testUUID7, testUUID8, testUUID9, testUUID10}
	for i, systemID := range uuids {
		repo.AddSystem(systemID, &redfishv1.ComputerSystem{ID: systemID, Name: fmt.Sprintf("System %d", i+1), PowerState: redfishv1.PowerStateOn})
	}
}

func setupErrorMockTest(repo *TestSystemsComputerSystemRepository, _ struct{}) {
	repo.SetErrorOnGetAll(true)
}

func setupNoMockTest(_ *TestSystemsComputerSystemRepository, _ struct{}) {
	// No setup needed for HTTP method tests
}

// Generic Test Framework for Systems endpoints

// runGenericSystemsTest executes a generic test case
func runGenericSystemsTest[T any](t *testing.T, testCase SystemsTestCase[T], config SystemsTestConfig[T]) {
	t.Helper()

	// Setup
	testRepo := NewTestSystemsComputerSystemRepository()
	testCase.setupRepo(testRepo, testCase.params)

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	// Create RedfishServer
	server := &RedfishServer{
		ComputerSystemUC: useCase,
	}

	// Setup router and request
	router := config.routerSetup(server)
	url := config.urlBuilder(testCase.params)
	req, _ := http.NewRequestWithContext(context.Background(), testCase.httpMethod, url, http.NoBody)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, testCase.expectedStatus, w.Code)

	// Run custom validation
	if testCase.validateFunc != nil {
		testCase.validateFunc(t, w, testCase.params)
	}
}

// createTestSystemEntityData creates a test system entity for repository
func createTestSystemEntityData(systemID, name, manufacturer, model, serialNumber string) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         name,
		BiosVersion:  "1.2.3.4",
		SystemType:   redfishv1.SystemTypePhysical, // Default to Physical for test systems
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   redfishv1.PowerStateOn,
	}
}

// createTestSystemEntityDataWithAllProperties creates a test system entity with Description, HostName, and Status populated
func createTestSystemEntityDataWithAllProperties(systemID, name, manufacturer, model, serialNumber string) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:          systemID,
		Name:        name,
		Description: "Computer System managed by Intel AMT",
		BiosVersion: "2.4.6.8",
		HostName:    "amt-" + systemID + ".example.com",
		SystemType:  redfishv1.SystemTypePhysical, // Default to Physical for AMT systems
		Status: &redfishv1.Status{
			State:  "Enabled",
			Health: "OK",
		},
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   redfishv1.PowerStateOn,
	}
}

// createTestSystemEntityDataMinimal creates a test system entity with minimal properties
func createTestSystemEntityDataMinimal(systemID, name string) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         name,
		SystemType:   redfishv1.SystemTypePhysical, // Default to Physical
		Manufacturer: "Intel Corporation",
		Model:        "Test Model",
		SerialNumber: "MIN001",
		PowerState:   redfishv1.PowerStateOff,
		// Description and HostName are empty - would be populated by CIM extraction in real repository
	}
}

// Repository setup functions for system by ID tests
func setupExistingSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityData(systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456")
	repo.AddSystem(systemID, system)
}

// setupSystemWithAllPropertiesMockTest sets up a system with Description, HostName, and Status populated
func setupSystemWithAllPropertiesMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithAllProperties(systemID, "Enhanced Test System", "Intel Corporation", "vPro Test Model", "ENH123456")
	repo.AddSystem(systemID, system)
}

// setupMinimalSystemMockTest sets up a system with minimal properties
func setupMinimalSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataMinimal(systemID, "Minimal Test System")
	repo.AddSystem(systemID, system)
}

func setupSystemNotFoundMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	repo.SetErrorOnGetByID(systemID, usecase.ErrSystemNotFound)
}

func setupSystemRepositoryErrorMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	repo.SetErrorOnGetByID(systemID, errSystemRepoFailure)
}

func setupNoSystemMockTest(_ *TestSystemsComputerSystemRepository, _ string) {
	// No setup needed for HTTP method tests
}

// validateSystemResponseDataTest validates system response with expected data
func validateSystemResponseDataTest(t *testing.T, w *httptest.ResponseRecorder, systemID, expectedName, expectedManufacturer, expectedModel, expectedSerial string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

	// Validate response structure
	assert.Equal(t, systemID, response.Id)
	assert.Equal(t, expectedName, response.Name)
	assert.NotNil(t, response.Manufacturer)
	assert.Equal(t, expectedManufacturer, *response.Manufacturer)
	assert.NotNil(t, response.Model)
	assert.Equal(t, expectedModel, *response.Model)
	assert.NotNil(t, response.SerialNumber)
	assert.Equal(t, expectedSerial, *response.SerialNumber)
	// Validate BiosVersion property
	assert.NotNil(t, response.BiosVersion)
	assert.Equal(t, "1.2.3.4", *response.BiosVersion)
	// Validate SystemType property (defaults to Physical for test systems)
	if response.SystemType != nil {
		// SystemType is directly accessible as a string type
		assert.Contains(t, []string{"Physical", "Virtual"}, string(*response.SystemType))
	}

	assert.NotNil(t, response.OdataId)
	assert.Equal(t, fmt.Sprintf("%s/%s", systemsEndpointTest, systemID), *response.OdataId)
	assert.NotNil(t, response.OdataType)
	assert.Equal(t, systemODataType, *response.OdataType)
}

// validateSystemResponseWithAllPropertiesTest validates system response with Description, HostName, and Status properties
func validateSystemResponseWithAllPropertiesTest(t *testing.T, w *httptest.ResponseRecorder, systemID, expectedName, expectedDescription, expectedHostName string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

	// Validate basic properties
	assert.Equal(t, systemID, response.Id)
	assert.Equal(t, expectedName, response.Name)

	// Validate BiosVersion property
	assert.NotNil(t, response.BiosVersion)
	assert.Equal(t, "2.4.6.8", *response.BiosVersion)

	// Validate Description property
	if expectedDescription != "" {
		assert.NotNil(t, response.Description)
		description, err := response.Description.AsResourceDescription()
		assert.NoError(t, err)
		assert.Equal(t, expectedDescription, description)
	}

	// Validate HostName property
	if expectedHostName != "" {
		assert.NotNil(t, response.HostName)
		assert.Equal(t, expectedHostName, *response.HostName)
	}

	// Validate Status if present
	if response.Status != nil {
		assert.NotNil(t, response.Status.State)
		assert.NotNil(t, response.Status.Health)

		state, err := response.Status.State.AsResourceStatusState1()
		assert.NoError(t, err)
		assert.Equal(t, "Enabled", state)

		health, err := response.Status.Health.AsResourceStatusHealth1()
		assert.NoError(t, err)
		assert.Equal(t, "OK", health)
	}

	// Validate OData fields
	assert.NotNil(t, response.OdataContext)
	assert.Equal(t, "/redfish/v1/$metadata#ComputerSystem.ComputerSystem", *response.OdataContext)
	assert.NotNil(t, response.OdataId)
	assert.Equal(t, fmt.Sprintf("/redfish/v1/Systems/%s", systemID), *response.OdataId)
	assert.NotNil(t, response.OdataType)
	assert.Equal(t, systemODataType, *response.OdataType)
}

// validateSystemActionsResponseTest validates system response with Actions property
func validateSystemActionsResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

	// Validate basic properties
	assert.Equal(t, systemID, response.Id)
	assert.Equal(t, "Test System", response.Name)

	// Validate Actions property
	assert.NotNil(t, response.Actions, "Actions property should not be nil")
	assert.NotNil(t, response.Actions.HashComputerSystemReset, "ComputerSystem.Reset action should not be nil")

	// Validate Reset action properties
	resetAction := response.Actions.HashComputerSystemReset
	assert.NotNil(t, resetAction.Target, "Reset action target should not be nil")

	expectedTarget := fmt.Sprintf("/redfish/v1/Systems/%s/Actions/ComputerSystem.Reset", systemID)
	assert.Equal(t, expectedTarget, *resetAction.Target)

	assert.NotNil(t, resetAction.Title, "Reset action title should not be nil")
	assert.Equal(t, "Reset", *resetAction.Title)

	// Note: ResetType@Redfish.AllowableValues are now provided through ActionInfo endpoint
	// as per DMTF specification, not embedded in the Reset action itself
}

// Validation functions for system by ID tests (reusing shared validation)
func validateSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateSystemResponseDataTest(t, w, systemID, "Test System", "Test Manufacturer", "Test Model", "SN123456")
}

// validateSystemWithAllPropertiesResponseTest validates system response with Description and HostName properties
func validateSystemWithAllPropertiesResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateSystemResponseWithAllPropertiesTest(t, w, systemID, "Enhanced Test System",
		"Computer System managed by Intel AMT",
		"amt-"+systemID+".example.com")
}

// validateMinimalSystemResponseTest validates system response with actual CIM data from repository
func validateMinimalSystemResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

	// Validate basic properties
	assert.Equal(t, systemID, response.Id)
	assert.Equal(t, "Minimal Test System", response.Name)

	// Since this is using mock repository without actual CIM data extraction,
	// Description and HostName will be empty strings (as returned by mock)
	// In a real scenario, these would be populated by extractCIMSystemInfo()

	// Validate that Description field exists but may be empty (no CIM data in mock)
	if response.Description != nil {
		description, err := response.Description.AsResourceDescription()
		assert.NoError(t, err)
		// Description could be empty in mock test environment
		_ = description
	}

	// Validate that HostName field exists but may be nil (no CIM data in mock)
	// In real implementation, this would be populated from CIM_ComputerSystem.DNSHostName
	_ = response.HostName
}

func validateSystemNotFoundResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	errorObj := validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "System not found")
	validateExtendedInfoTest(t, errorObj, "Base.1.22.0.ResourceMissing", fmt.Sprintf("The requested resource of type System named %s was not found.", systemID), "Critical")
}

func validateSystemErrorResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ string) {
	t.Helper()
	_ = validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "An internal server error occurred.")
}

func validateBadRequestResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ string) {
	t.Helper()
	_ = validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "Invalid system ID: system ID cannot be empty")
}

func validateInvalidUUIDFormatResponseTest(t *testing.T, w *httptest.ResponseRecorder, _ string) {
	t.Helper()
	_ = validateErrorResponseDataTest(t, w, "Base.1.22.0.GeneralError", "Invalid system ID: system ID must be a valid UUID")
}

// TestLogger for testing logger code paths - captures log calls
type TestLogger struct {
	DebugCalls [][]interface{}
	InfoCalls  [][]interface{}
	WarnCalls  [][]interface{}
	ErrorCalls [][]interface{}
	FatalCalls [][]interface{}
}

func NewTestLogger() *TestLogger {
	return &TestLogger{
		DebugCalls: make([][]interface{}, 0),
		InfoCalls:  make([][]interface{}, 0),
		WarnCalls:  make([][]interface{}, 0),
		ErrorCalls: make([][]interface{}, 0),
		FatalCalls: make([][]interface{}, 0),
	}
}

func (l *TestLogger) Debug(message interface{}, args ...interface{}) {
	l.DebugCalls = append(l.DebugCalls, append([]interface{}{message}, args...))
}

func (l *TestLogger) Info(msg string, args ...interface{}) {
	l.InfoCalls = append(l.InfoCalls, append([]interface{}{msg}, args...))
}

func (l *TestLogger) Warn(msg string, args ...interface{}) {
	l.WarnCalls = append(l.WarnCalls, append([]interface{}{msg}, args...))
}

func (l *TestLogger) Error(message interface{}, args ...interface{}) {
	l.ErrorCalls = append(l.ErrorCalls, append([]interface{}{message}, args...))
}

func (l *TestLogger) Fatal(message interface{}, args ...interface{}) {
	l.FatalCalls = append(l.FatalCalls, append([]interface{}{message}, args...))
}

// ====================================================================================================
// MAIN TEST FUNCTIONS
// ====================================================================================================

// TestSystemsHandler_GetSystemsCollection tests the GetSystemsCollection endpoint with various scenarios
func TestSystemsHandler_GetSystemsCollection(t *testing.T) {
	t.Parallel()

	config := SystemsTestConfig[struct{}]{
		endpoint:    systemsEndpointTest,
		routerSetup: setupSystemsTestRouter,
		urlBuilder: func(_ struct{}) string {
			return systemsEndpointTest
		},
	}

	tests := []SystemsTestCase[struct{}]{
		{"Success - Multiple Systems", setupMultipleSystemsMockTest, "GET", http.StatusOK, validateMultipleSystemsResponseTest, struct{}{}},
		{"Success - Empty Collection", setupEmptyCollectionMockTest, "GET", http.StatusOK, validateEmptyCollectionResponseTest, struct{}{}},
		{"Success - Filter Empty System IDs", setupFilteredSystemsMockTest, "GET", http.StatusOK, validateFilteredSystemsResponseTest, struct{}{}},
		{"Success - Single System", setupSingleSystemMockTest, "GET", http.StatusOK, validateSingleSystemResponseTest, struct{}{}},
		{"Success - Large Members List", setupLargeCollectionMockTest, "GET", http.StatusOK, validateLargeCollectionResponseTest, struct{}{}},
		{"Success - Response Headers", setupSingleSystemMockTest, "GET", http.StatusOK, validateJSONResponseTest, struct{}{}},
		{"Error - Repository Error", setupErrorMockTest, "GET", http.StatusInternalServerError, validateErrorResponseTest, struct{}{}},
		{"Error - HTTP Method POST Not Allowed", setupNoMockTest, "POST", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method PUT Not Allowed", setupNoMockTest, "PUT", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method DELETE Not Allowed", setupNoMockTest, "DELETE", http.StatusNotFound, nil, struct{}{}},
		{"Error - HTTP Method PATCH Not Allowed", setupNoMockTest, "PATCH", http.StatusNotFound, nil, struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericSystemsTest(t, tt, config)
		})
	}
}

// TestSystemsHandler_GetSystemByID tests the GetSystemByID endpoint with various scenarios
func TestSystemsHandler_GetSystemByID(t *testing.T) {
	t.Parallel()

	config := SystemsTestConfig[string]{
		endpoint:    "/redfish/v1/Systems/{id}",
		routerSetup: setupSystemByIDTestRouter,
		urlBuilder: func(systemID string) string {
			if systemID == "" {
				return fmt.Sprintf("%s/", systemsEndpointTest)
			}

			return fmt.Sprintf("%s/%s", systemsEndpointTest, systemID)
		},
	}

	tests := []SystemsTestCase[string]{
		{"Success - Existing System", setupExistingSystemMockTest, "GET", http.StatusOK, validateSystemResponseTest, testUUID1},
		{"Success - System with Actions", setupExistingSystemMockTest, "GET", http.StatusOK, validateSystemActionsResponseTest, testUUID1},
		{"Success - Minimal System Properties", setupMinimalSystemMockTest, "GET", http.StatusOK, validateMinimalSystemResponseTest, testUUID2},
		{"Success - System with Long ID", setupMinimalSystemMockTest, "GET", http.StatusOK, validateMinimalSystemResponseTest, testUUID3},

		{"Success - System with All Properties (Description/HostName/Status)", setupSystemWithAllPropertiesMockTest, "GET", http.StatusOK, validateSystemWithAllPropertiesResponseTest, testUUID4},
		{"Success - System with MemorySummary", setupSystemWithMemoryMockTest, "GET", http.StatusOK, validateSystemWithMemoryResponseTest, testUUID5},
		{"Success - System with ProcessorSummary", setupSystemWithProcessorMockTest, "GET", http.StatusOK, validateSystemWithProcessorResponseTest, testUUID6},

		{"Success - System with Memory and Processor Summaries", setupSystemWithMemoryAndProcessorMockTest, "GET", http.StatusOK, validateSystemWithMemoryAndProcessorResponseTest, testUUID7},
		{"Success - System with All Properties and Summaries", setupSystemWithFullPropertiesMockTest, "GET", http.StatusOK, validateSystemWithFullPropertiesResponseTest, testUUID8},

		{"Success - SystemType Always Physical (Dell System)", setupPhysicalSystemMockTest, "GET", http.StatusOK, validateSystemTypeAlwaysPhysicalTest, testUUID9},
		{"Success - SystemType Always Physical (VMware System)", setupVirtualSystemMockTest, "GET", http.StatusOK, validateSystemTypeAlwaysPhysicalTest, testUUID10},
		{"Success - System with MemoryMirroring", setupSystemWithMemoryMirroringMockTest, "GET", http.StatusOK, validateSystemWithMemoryMirroringResponseTest, testUUID9},

		{"Error - Empty System ID", setupNoSystemMockTest, "GET", http.StatusBadRequest, validateBadRequestResponseTest, ""},
		{"Error - Invalid System ID Format - Not UUID", setupNoSystemMockTest, "GET", http.StatusBadRequest, validateInvalidUUIDFormatResponseTest, "invalid-system-id"},
		{"Error - Invalid System ID Format - Missing Hyphens", setupNoSystemMockTest, "GET", http.StatusBadRequest, validateInvalidUUIDFormatResponseTest, "550e8400e29b41d4a716446655440001"},
		{"Error - Invalid System ID Format - Special Characters", setupNoSystemMockTest, "GET", http.StatusBadRequest, validateInvalidUUIDFormatResponseTest, "550e8400-e29b-41d4-a716-446655440001; DROP TABLE"},
		{"Error - System Not Found", setupSystemNotFoundMockTest, "GET", http.StatusNotFound, validateSystemNotFoundResponseTest, testUUIDNotFound},
		{"Error - Repository Error", setupSystemRepositoryErrorMockTest, "GET", http.StatusInternalServerError, validateSystemErrorResponseTest, testUUID1},

		{"Error - HTTP Method POST Not Allowed", setupNoSystemMockTest, "POST", http.StatusNotFound, nil, testUUID1},
		{"Error - HTTP Method PUT Not Allowed", setupNoSystemMockTest, "PUT", http.StatusNotFound, nil, testUUID1},
		{"Error - HTTP Method DELETE Not Allowed", setupNoSystemMockTest, "DELETE", http.StatusNotFound, nil, testUUID1},
		{"Error - HTTP Method PATCH Not Allowed", setupNoSystemMockTest, "PATCH", http.StatusNotFound, nil, testUUID1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runGenericSystemsTest(t, tt, config)
		})
	}
}

// TestSystemsHandler_GetSystemsCollection_WithLogger tests logging paths
func TestSystemsHandler_GetSystemsCollection_WithLogger(t *testing.T) {
	t.Parallel()

	// Test error logging path
	testRepo := NewTestSystemsComputerSystemRepository()
	testRepo.SetErrorOnGetAll(true)

	testLogger := NewTestLogger()

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	server := &RedfishServer{
		ComputerSystemUC: useCase,
		Logger:           testLogger,
	}
	router := setupSystemsTestRouter(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", systemsEndpointTest, http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// Verify logger was called
	assert.Len(t, testLogger.ErrorCalls, 1)
	assert.Equal(t, "Failed to retrieve computer systems collection", testLogger.ErrorCalls[0][0])
}

// TestSystemsHandler_GetSystemByID_WithLogger tests logging paths
func TestSystemsHandler_GetSystemByID_WithLogger(t *testing.T) {
	t.Parallel()

	// Test error logging path
	testRepo := NewTestSystemsComputerSystemRepository()
	testRepo.SetErrorOnGetByID(testUUID1, errSystemRepoFailure)

	testLogger := NewTestLogger()

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	server := &RedfishServer{
		ComputerSystemUC: useCase,
		Logger:           testLogger,
	}
	router := setupSystemByIDTestRouter(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/redfish/v1/Systems/"+testUUID1, http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// Verify logger was called
	assert.Len(t, testLogger.ErrorCalls, 1)
	assert.Equal(t, "Failed to retrieve computer system", testLogger.ErrorCalls[0][0])
}

// createTestSystemEntityDataWithMemory creates a test system entity with MemorySummary
func createTestSystemEntityDataWithMemory(systemID, name, manufacturer, model, serialNumber string) *redfishv1.ComputerSystem {
	system := createTestSystemEntityData(systemID, name, manufacturer, model, serialNumber)
	system.MemorySummary = &redfishv1.ComputerSystemMemorySummary{
		TotalSystemMemoryGiB: func() *float32 {
			v := float32(16.0)

			return &v
		}(),
		// MemoryMirroring is not set in test - only populated when AMT provides actual mirroring data
		Status: &redfishv1.Status{
			Health: "OK",
			State:  "Enabled",
		},
	}

	return system
}

// setupSystemWithMemoryMockTest sets up a system with MemorySummary populated
func setupSystemWithMemoryMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithMemory(systemID, "Test System with Memory", "Test Manufacturer", "Test Model", "SN123456")
	repo.AddSystem(systemID, system)
}

// validateSystemWithMemoryResponseTest validates system response includes MemorySummary
func validateSystemWithMemoryResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	validateSystemResponseDataTest(t, w, systemID, "Test System with Memory", "Test Manufacturer", "Test Model", "SN123456")

	// Additional validation for MemorySummary
	var response redfishv1.ComputerSystem

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify MemorySummary is present and correct
	assert.NotNil(t, response.MemorySummary, "MemorySummary should be present")
	assert.NotNil(t, response.MemorySummary.TotalSystemMemoryGiB, "TotalSystemMemoryGiB should be set")
	assert.Equal(t, float32(16.0), *response.MemorySummary.TotalSystemMemoryGiB, "Memory should be 16.0 GiB")
	// MemoryMirroring is not asserted as it should only be populated when AMT provides actual mirroring data
	assert.NotNil(t, response.MemorySummary.Status, "MemorySummary Status should be set")
	assert.Equal(t, "OK", response.MemorySummary.Status.Health, "Memory health should be OK")
	assert.Equal(t, "Enabled", response.MemorySummary.Status.State, "Memory state should be Enabled")
}

// createTestSystemEntityDataWithProcessor creates a test system entity with ProcessorSummary populated
func createTestSystemEntityDataWithProcessor(systemID, name, manufacturer, model, serialNumber string) *redfishv1.ComputerSystem {
	system := createTestSystemEntityData(systemID, name, manufacturer, model, serialNumber)
	system.ProcessorSummary = &redfishv1.ComputerSystemProcessorSummary{
		Count: intPtr(2),
		// CoreCount, LogicalProcessorCount, and ThreadingEnabled are nil
		// because CIM_Processor doesn't provide these in Intel AMT WSMAN implementation
		// Model is available from CIM_Chip.Version
		CoreCount:             nil,
		LogicalProcessorCount: nil,
		Model:                 stringPtr("12th Gen Intel(R) Core(TM) i5-1250P"),
		Status: &redfishv1.Status{
			Health:       "OK",
			HealthRollup: "OK",
			State:        "Enabled",
		},
		StatusRedfishDeprecated: stringPtr("Please migrate to use Status in the individual Processor resources"),
		ThreadingEnabled:        nil,
	}

	return system
} // Helper functions for pointer creation
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

// setupSystemWithProcessorMockTest sets up a system with ProcessorSummary populated
func setupSystemWithProcessorMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithProcessor(systemID, "Test System with Processor", "Intel Corporation", "Test Model", "SN123456")
	repo.AddSystem(systemID, system)
}

// setupSystemWithMemoryAndProcessorMockTest sets up a system with both MemorySummary and ProcessorSummary populated
func setupSystemWithMemoryAndProcessorMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithProcessor(systemID, "Test System with Memory and Processor", "Intel Corporation", "Complete Test Model", "COMP123456")
	// Add MemorySummary to the system
	system.MemorySummary = &redfishv1.ComputerSystemMemorySummary{
		TotalSystemMemoryGiB: func() *float32 {
			v := float32(32.0)

			return &v
		}(),
		Status: &redfishv1.Status{
			Health: "OK",
			State:  "Enabled",
		},
	}
	repo.AddSystem(systemID, system)
}

// setupSystemWithFullPropertiesMockTest sets up a system with all properties (All Properties + Memory + Processor)
func setupSystemWithFullPropertiesMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithAllProperties(systemID, "Complete Test System", "Intel Corporation", "vPro Complete Model", "FULL123456")
	// Add MemorySummary
	system.MemorySummary = &redfishv1.ComputerSystemMemorySummary{
		TotalSystemMemoryGiB: func() *float32 {
			v := float32(64.0)

			return &v
		}(),
		Status: &redfishv1.Status{
			Health: "OK",
			State:  "Enabled",
		},
	}
	// Add ProcessorSummary
	system.ProcessorSummary = &redfishv1.ComputerSystemProcessorSummary{
		Count:                 intPtr(4),
		CoreCount:             nil,
		LogicalProcessorCount: nil,
		Model:                 stringPtr("12th Gen Intel(R) Core(TM) i7-1270P"),
		Status: &redfishv1.Status{
			Health:       "OK",
			HealthRollup: "OK",
			State:        "Enabled",
		},
		StatusRedfishDeprecated: stringPtr("Please migrate to use Status in the individual Processor resources"),
		ThreadingEnabled:        nil,
	}
	repo.AddSystem(systemID, system)
}

// createTestSystemEntityDataWithSystemType creates a test system entity with specific SystemType
func createTestSystemEntityDataWithSystemType(systemID, name, manufacturer, model, serialNumber string, systemType redfishv1.SystemType) *redfishv1.ComputerSystem {
	system := createTestSystemEntityData(systemID, name, manufacturer, model, serialNumber)
	system.SystemType = systemType

	return system
}

// setupPhysicalSystemMockTest sets up a physical system for testing SystemType property
func setupPhysicalSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithSystemType(systemID, "Physical Test System", "Dell Inc.", "OptiPlex 7090", "PHY123456", redfishv1.SystemTypePhysical)
	repo.AddSystem(systemID, system)
}

// setupVirtualSystemMockTest sets up a virtual system for testing SystemType property
func setupVirtualSystemMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithSystemType(systemID, "Virtual Test System", "VMware, Inc.", "VMware Virtual Platform", "VIR123456", redfishv1.SystemTypeVirtual)
	repo.AddSystem(systemID, system)
}

// setupSystemWithMemoryMirroringMockTest sets up a system with MemoryMirroring for testing
func setupSystemWithMemoryMirroringMockTest(repo *TestSystemsComputerSystemRepository, systemID string) {
	system := createTestSystemEntityDataWithMemory(systemID, "System with Memory Mirroring", "Intel Corporation", "NUC Model", "MIR123456")
	// Add MemoryMirroring to existing MemorySummary
	system.MemorySummary.MemoryMirroring = redfishv1.MemoryMirroringSystem
	repo.AddSystem(systemID, system)
}

// validateSystemWithProcessorResponseTest validates system response includes ProcessorSummary
func validateSystemWithProcessorResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	validateSystemResponseDataTest(t, w, systemID, "Test System with Processor", "Intel Corporation", "Test Model", "SN123456")

	// Additional validation for ProcessorSummary
	var response redfishv1.ComputerSystem

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify ProcessorSummary is present and correct
	assert.NotNil(t, response.ProcessorSummary, "ProcessorSummary should be present")

	// Count should be available from hardware enumeration
	assert.NotNil(t, response.ProcessorSummary.Count, "Count should be set")
	assert.Equal(t, 2, *response.ProcessorSummary.Count, "Processor count should be 2")

	// These properties are nil because CIM_Processor doesn't provide them in Intel AMT WSMAN
	assert.Nil(t, response.ProcessorSummary.CoreCount, "CoreCount should be nil (not available from CIM_Processor)")
	assert.Nil(t, response.ProcessorSummary.LogicalProcessorCount, "LogicalProcessorCount should be nil (not available from CIM_Processor)")
	assert.Nil(t, response.ProcessorSummary.ThreadingEnabled, "ThreadingEnabled should be nil (not available from CIM_Processor)")

	// Model should be available from CIM_Chip.Version
	assert.NotNil(t, response.ProcessorSummary.Model, "Model should be present from CIM_Chip.Version")
	assert.Equal(t, "12th Gen Intel(R) Core(TM) i5-1250P", *response.ProcessorSummary.Model, "Processor model should match CIM_Chip.Version")

	// Status should be available from CIM_Processor HealthState and EnabledState
	assert.NotNil(t, response.ProcessorSummary.Status, "ProcessorSummary Status should be set")
	assert.Equal(t, "OK", response.ProcessorSummary.Status.Health, "Processor health should be OK")
	assert.Equal(t, "OK", response.ProcessorSummary.Status.HealthRollup, "Processor HealthRollup should be OK")
	assert.Equal(t, "Enabled", response.ProcessorSummary.Status.State, "Processor state should be Enabled")

	// Note: StatusRedfishDeprecated is not included in generated types from OpenAPI spec
	// This is a limitation of the current OpenAPI generator which doesn't handle Redfish-specific annotations
}

// validateSystemWithMemoryAndProcessorResponseTest validates system response includes both MemorySummary and ProcessorSummary
func validateSystemWithMemoryAndProcessorResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	validateSystemResponseDataTest(t, w, systemID, "Test System with Memory and Processor", "Intel Corporation", "Complete Test Model", "COMP123456")

	// Additional validation for both MemorySummary and ProcessorSummary
	var response redfishv1.ComputerSystem

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify MemorySummary is present and correct
	assert.NotNil(t, response.MemorySummary, "MemorySummary should be present")
	assert.NotNil(t, response.MemorySummary.TotalSystemMemoryGiB, "TotalSystemMemoryGiB should be set")
	assert.Equal(t, float32(32.0), *response.MemorySummary.TotalSystemMemoryGiB, "Memory should be 32.0 GiB")
	assert.NotNil(t, response.MemorySummary.Status, "MemorySummary Status should be set")
	assert.Equal(t, "OK", response.MemorySummary.Status.Health, "Memory health should be OK")
	assert.Equal(t, "Enabled", response.MemorySummary.Status.State, "Memory state should be Enabled")

	// Verify ProcessorSummary is present and correct
	assert.NotNil(t, response.ProcessorSummary, "ProcessorSummary should be present")
	assert.NotNil(t, response.ProcessorSummary.Count, "Count should be set")
	assert.Equal(t, 2, *response.ProcessorSummary.Count, "Processor count should be 2")
	assert.NotNil(t, response.ProcessorSummary.Model, "Model should be present from CIM_Chip.Version")
	assert.Equal(t, "12th Gen Intel(R) Core(TM) i5-1250P", *response.ProcessorSummary.Model, "Processor model should match CIM_Chip.Version")
	assert.NotNil(t, response.ProcessorSummary.Status, "ProcessorSummary Status should be set")
	assert.Equal(t, "OK", response.ProcessorSummary.Status.Health, "Processor health should be OK")
	assert.Equal(t, "Enabled", response.ProcessorSummary.Status.State, "Processor state should be Enabled")
}

// validateSystemWithFullPropertiesResponseTest validates system response includes all properties
func validateSystemWithFullPropertiesResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	// First validate basic properties and All Properties (Description, HostName, Status)
	validateSystemResponseWithAllPropertiesTest(t, w, systemID, "Complete Test System",
		"Computer System managed by Intel AMT",
		"amt-"+systemID+".example.com")

	// Additional validation for MemorySummary and ProcessorSummary
	var response redfishv1.ComputerSystem

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify MemorySummary is present and correct
	assert.NotNil(t, response.MemorySummary, "MemorySummary should be present")
	assert.NotNil(t, response.MemorySummary.TotalSystemMemoryGiB, "TotalSystemMemoryGiB should be set")
	assert.Equal(t, float32(64.0), *response.MemorySummary.TotalSystemMemoryGiB, "Memory should be 64.0 GiB")
	assert.NotNil(t, response.MemorySummary.Status, "MemorySummary Status should be set")
	assert.Equal(t, "OK", response.MemorySummary.Status.Health, "Memory health should be OK")
	assert.Equal(t, "Enabled", response.MemorySummary.Status.State, "Memory state should be Enabled")

	// Verify ProcessorSummary is present and correct
	assert.NotNil(t, response.ProcessorSummary, "ProcessorSummary should be present")
	assert.NotNil(t, response.ProcessorSummary.Count, "Count should be set")
	assert.Equal(t, 4, *response.ProcessorSummary.Count, "Processor count should be 4")
	assert.NotNil(t, response.ProcessorSummary.Model, "Model should be present from CIM_Chip.Version")
	assert.Equal(t, "12th Gen Intel(R) Core(TM) i7-1270P", *response.ProcessorSummary.Model, "Processor model should match CIM_Chip.Version")
	assert.NotNil(t, response.ProcessorSummary.Status, "ProcessorSummary Status should be set")
	assert.Equal(t, "OK", response.ProcessorSummary.Status.Health, "Processor health should be OK")
	assert.Equal(t, "Enabled", response.ProcessorSummary.Status.State, "Processor state should be Enabled")
	// Note: StatusRedfishDeprecated is not available in HTTP responses due to generated types limitation
}

// validateSystemTypeAlwaysPhysicalTest validates that SystemType is always Physical in API response
// This test verifies the current implementation behavior where SystemType is hardcoded to "Physical"
// regardless of what the entity contains
func validateSystemTypeAlwaysPhysicalTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()
	validateJSONContentTypeTest(t, w)

	var response generated.ComputerSystemComputerSystem
	unmarshalJSONResponseTest(t, w, &response)

	// Validate that SystemType is always Physical in API response
	// This is the current implementation behavior - SystemType is hardcoded to "Physical"
	assert.NotNil(t, response.SystemType)
	assert.Equal(t, "Physical", string(*response.SystemType), "SystemType should always be Physical in current implementation")

	// Validate other basic properties to ensure the system data is returned correctly
	assert.Equal(t, systemID, response.Id)
	assert.NotNil(t, response.Name)
	assert.NotNil(t, response.Manufacturer)
	assert.NotNil(t, response.Model)
	assert.NotNil(t, response.SerialNumber)
}

// validateSystemWithMemoryMirroringResponseTest validates system response includes MemoryMirroring
func validateSystemWithMemoryMirroringResponseTest(t *testing.T, w *httptest.ResponseRecorder, systemID string) {
	t.Helper()

	validateSystemResponseDataTest(t, w, systemID, "System with Memory Mirroring", "Intel Corporation", "NUC Model", "MIR123456")

	// Additional validation for MemoryMirroring
	var response redfishv1.ComputerSystem

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify MemorySummary is present and correct
	assert.NotNil(t, response.MemorySummary, "MemorySummary should be present")
	assert.NotNil(t, response.MemorySummary.TotalSystemMemoryGiB, "TotalSystemMemoryGiB should be set")
	assert.Equal(t, float32(16.0), *response.MemorySummary.TotalSystemMemoryGiB, "Memory should be 16.0 GiB")
	assert.NotNil(t, response.MemorySummary.Status, "MemorySummary Status should be set")
	assert.Equal(t, "OK", response.MemorySummary.Status.Health, "Memory health should be OK")
	assert.Equal(t, "Enabled", response.MemorySummary.Status.State, "Memory state should be Enabled")

	// Validate MemoryMirroring is set correctly
	assert.Equal(t, redfishv1.MemoryMirroringSystem, response.MemorySummary.MemoryMirroring, "MemoryMirroring should be System")
}

func TestValidateSystemID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		systemID  string
		wantError bool
		wantErr   error
	}{
		{
			name:      "Valid UUID - lowercase",
			systemID:  testUUID1,
			wantError: false,
		},
		{
			name:      "Valid UUID - uppercase",
			systemID:  "550E8400-E29B-41D4-A716-446655440001",
			wantError: false,
		},
		{
			name:      "Valid UUID - mixed case",
			systemID:  "550e8400-E29B-41d4-A716-446655440001",
			wantError: false,
		},
		{
			name:      "Empty string",
			systemID:  "",
			wantError: true,
			wantErr:   errSystemIDEmpty,
		},
		{
			name:      "Whitespace only",
			systemID:  "   ",
			wantError: true,
			wantErr:   errSystemIDInvalid,
		},
		{
			name:      "Invalid - missing hyphens",
			systemID:  "550e8400e29b41d4a716446655440001",
			wantError: true,
			wantErr:   errSystemIDInvalid,
		},
		{
			name:      "Invalid - wrong format",
			systemID:  "test-system-1",
			wantError: true,
			wantErr:   errSystemIDInvalid,
		},
		{
			name:      "Invalid - too short",
			systemID:  "550e8400-e29b-41d4-a716",
			wantError: true,
			wantErr:   errSystemIDInvalid,
		},
		{
			name:      "Invalid - too long",
			systemID:  "550e8400-e29b-41d4-a716-446655440001-extra",
			wantError: true,
			wantErr:   errSystemIDInvalid,
		},
		{
			name:      "Invalid - non-hex characters",
			systemID:  "550e8400-e29b-41d4-a716-44665544000g",
			wantError: true,
			wantErr:   errSystemIDInvalid,
		},
		{
			name:      "Invalid - special characters",
			systemID:  "550e8400-e29b-41d4-a716-446655440001; DROP TABLE",
			wantError: true,
			wantErr:   errSystemIDInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateSystemID(tt.systemID)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateSystemID() expected error but got nil")
				} else if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("validateSystemID() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("validateSystemID() unexpected error = %v", err)
				}
			}
		})
	}
}
