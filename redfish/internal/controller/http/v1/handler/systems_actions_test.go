package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

var configTestMu sync.Mutex

// Test constants for system actions
const (
	testSystemID                = "550e8400-e29b-41d4-a716-446655440001"
	resetActionEndpoint         = "/redfish/v1/Systems/550e8400-e29b-41d4-a716-446655440001/Actions/ComputerSystem.Reset"
	generateTokenActionEndpoint = "/redfish/v1/Systems/550e8400-e29b-41d4-a716-446655440001/Actions/Oem/IntelComputerSystem.GenerateRedirectionToken"
	taskServiceEndpoint         = "/redfish/v1/TaskService/Tasks/"
	taskODataContext            = "/redfish/v1/$metadata#Task.Task"
	taskODataType               = "#Task.v1_6_0.Task"
)

// setupSystemActionsTestServer creates a test server with a mock repository
func setupSystemActionsTestServer(repo *TestSystemsComputerSystemRepository) *RedfishServer {
	uc := &usecase.ComputerSystemUseCase{
		Repo: repo,
	}

	return &RedfishServer{
		ComputerSystemUC: uc,
	}
}

// setupSystemActionsTestRouter sets up a gin router for testing
func setupSystemActionsTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/redfish/v1/Systems/:computerSystemId/Actions/ComputerSystem.Reset",
		func(c *gin.Context) {
			computerSystemID := c.Param("computerSystemId")
			server.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset(c, computerSystemID)
		})
	router.POST("/redfish/v1/Systems/:computerSystemId/Actions/Oem/IntelComputerSystem.GenerateRedirectionToken",
		func(c *gin.Context) {
			computerSystemID := c.Param("computerSystemId")
			server.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionToken(c, computerSystemID)
		})

	return router
}

// createResetRequest creates a JSON request body for reset action
func createResetRequest(resetType generated.ResourceResetType) []byte {
	req := generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody{
		ResetType: &resetType,
	}
	body, _ := json.Marshal(req)

	return body
}

// executeResetRequest executes a reset request and returns the response recorder
func executeResetRequest(router *gin.Engine, endpoint string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func createEmptyActionRequest() []byte {
	return []byte(`{}`)
}

func configureTestJWT(t *testing.T) {
	t.Helper()

	configTestMu.Lock()

	originalConfig := config.ConsoleConfig
	config.ConsoleConfig = &config.Config{
		Auth: config.Auth{
			JWTKey:                   "redfish-test-jwt-key",
			JWTExpiration:            time.Hour,
			RedirectionJWTExpiration: 5 * time.Minute,
		},
	}

	t.Cleanup(func() {
		config.ConsoleConfig = originalConfig
		configTestMu.Unlock()
	})
}

// assertErrorResponse verifies the response contains an error
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var errorResponse map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse, "error")
}

// assertTaskResponse verifies the task response structure
func assertTaskResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var taskResponse map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &taskResponse)
	assert.NoError(t, err)

	assert.Equal(t, taskODataContext, taskResponse["@odata.context"])
	assert.Equal(t, taskODataType, taskResponse["@odata.type"])
	assert.Contains(t, taskResponse["@odata.id"], taskServiceEndpoint)
	assert.Equal(t, "Completed", taskResponse["TaskState"])
	assert.Equal(t, "OK", taskResponse["TaskStatus"])
	assert.NotEmpty(t, taskResponse["Id"])
	assert.NotEmpty(t, taskResponse["StartTime"])
	assert.NotEmpty(t, taskResponse["EndTime"])

	location := w.Header().Get("Location")
	assert.Contains(t, location, taskServiceEndpoint)
}

func TestPostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset_Success(t *testing.T) {
	t.Parallel()

	// Test cases for successful reset operations
	testCases := []struct {
		name          string
		resetType     generated.ResourceResetType
		initialState  redfishv1.PowerState
		expectedState redfishv1.PowerState
	}{
		{
			name:          "GracefulShutdown from On",
			resetType:     generated.ResourceResetTypeGracefulShutdown,
			initialState:  redfishv1.PowerStateOn,
			expectedState: redfishv1.PowerStateOff,
		},
		{
			name:          "ForceOff from On",
			resetType:     generated.ResourceResetTypeForceOff,
			initialState:  redfishv1.PowerStateOn,
			expectedState: redfishv1.PowerStateOff,
		},
		{
			name:          "On from Off",
			resetType:     generated.ResourceResetTypeOn,
			initialState:  redfishv1.PowerStateOff,
			expectedState: redfishv1.PowerStateOn,
		},
		{
			name:          "ForceRestart from On",
			resetType:     generated.ResourceResetTypeForceRestart,
			initialState:  redfishv1.PowerStateOn,
			expectedState: redfishv1.PowerStateOff,
		},
		{
			name:          "GracefulRestart from On",
			resetType:     generated.ResourceResetTypeGracefulRestart,
			initialState:  redfishv1.PowerStateOn,
			expectedState: redfishv1.PowerStateOff,
		},
		{
			name:          "PowerCycle from On",
			resetType:     generated.ResourceResetTypePowerCycle,
			initialState:  redfishv1.PowerStateOn,
			expectedState: redfishv1.PowerStateOff,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup isolated repo and system for each subtest to avoid data races
			repo := NewTestSystemsComputerSystemRepository()
			system := &redfishv1.ComputerSystem{
				ID:           testSystemID,
				Name:         "Test System",
				PowerState:   tc.initialState,
				Manufacturer: "TestMfg",
				Model:        "TestModel",
				SerialNumber: "SN12345",
			}
			repo.AddSystem(testSystemID, system)

			server := setupSystemActionsTestServer(repo)
			router := setupSystemActionsTestRouter(server)

			body := createResetRequest(tc.resetType)
			w := executeResetRequest(router, resetActionEndpoint, body)

			assert.Equal(t, http.StatusAccepted, w.Code)
			assertTaskResponse(t, w)

			updatedSystem, err := repo.GetByID(context.Background(), testSystemID)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedState, updatedSystem.PowerState)
		})
	}
}

func TestPostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset_MalformedJSON(t *testing.T) {
	t.Parallel()

	repo := NewTestSystemsComputerSystemRepository()
	server := setupSystemActionsTestServer(repo)
	router := setupSystemActionsTestRouter(server)

	invalidJSON := []byte(`{"ResetType": invalid}`)
	w := executeResetRequest(router, resetActionEndpoint, invalidJSON)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorResponse(t, w)
}

func TestPostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset_MissingResetType(t *testing.T) {
	t.Parallel()

	repo := NewTestSystemsComputerSystemRepository()
	server := setupSystemActionsTestServer(repo)
	router := setupSystemActionsTestRouter(server)

	emptyReq := []byte(`{}`)
	w := executeResetRequest(router, resetActionEndpoint, emptyReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorResponse(t, w)
}

func TestPostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset_SystemNotFound(t *testing.T) {
	t.Parallel()

	repo := NewTestSystemsComputerSystemRepository()
	server := setupSystemActionsTestServer(repo)
	router := setupSystemActionsTestRouter(server)

	body := createResetRequest(generated.ResourceResetTypeOn)
	endpoint := "/redfish/v1/Systems/999e8400-e29b-41d4-a716-446655440000/Actions/ComputerSystem.Reset"
	w := executeResetRequest(router, endpoint, body)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assertErrorResponse(t, w)
}

func TestPostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset_InvalidResetType(t *testing.T) {
	t.Parallel()

	repo := NewTestSystemsComputerSystemRepository()
	system := &redfishv1.ComputerSystem{
		ID:         testSystemID,
		PowerState: redfishv1.PowerStateOn,
	}
	repo.AddSystem(testSystemID, system)

	server := setupSystemActionsTestServer(repo)
	router := setupSystemActionsTestRouter(server)

	invalidResetType := generated.ResourceResetType("InvalidResetType")
	body := createResetRequest(invalidResetType)
	w := executeResetRequest(router, resetActionEndpoint, body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorResponse(t, w)
}

func TestPostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset_EmptyResetType(t *testing.T) {
	t.Parallel()

	repo := NewTestSystemsComputerSystemRepository()
	server := setupSystemActionsTestServer(repo)
	router := setupSystemActionsTestRouter(server)

	emptyResetType := generated.ResourceResetType("")
	body := createResetRequest(emptyResetType)
	w := executeResetRequest(router, resetActionEndpoint, body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionToken_Success(t *testing.T) {
	t.Parallel()
	configureTestJWT(t)

	repo := NewTestSystemsComputerSystemRepository()
	repo.AddSystem(testSystemID, &redfishv1.ComputerSystem{ID: testSystemID, Name: "Test System"})

	server := setupSystemActionsTestServer(repo)
	router := setupSystemActionsTestRouter(server)

	w := executeResetRequest(router, generateTokenActionEndpoint, createEmptyActionRequest())

	assert.Equal(t, http.StatusOK, w.Code)

	var response generated.GenerateRedirectionTokenResponse

	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.RedirectionToken)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), response.ExpirationTime, 5*time.Second)

	parsedToken, err := jwt.Parse(response.RedirectionToken, func(_ *jwt.Token) (interface{}, error) {
		return []byte(config.ConsoleConfig.JWTKey), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, contentTypeJSON, w.Header().Get("Content-Type"))
	assert.Equal(t, SupportedODataVersion, w.Header().Get("OData-Version"))
}

func TestPostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionToken_SystemNotFound(t *testing.T) {
	t.Parallel()
	configureTestJWT(t)

	repo := NewTestSystemsComputerSystemRepository()
	server := setupSystemActionsTestServer(repo)
	router := setupSystemActionsTestRouter(server)

	endpoint := "/redfish/v1/Systems/999e8400-e29b-41d4-a716-446655440000/Actions/Oem/IntelComputerSystem.GenerateRedirectionToken"
	w := executeResetRequest(router, endpoint, createEmptyActionRequest())

	assert.Equal(t, http.StatusNotFound, w.Code)
	assertErrorResponse(t, w)
}
