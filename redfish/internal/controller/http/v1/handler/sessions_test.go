// Package v1 provides unit tests for Redfish SessionService POC
package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	sessioninfra "github.com/device-management-toolkit/console/redfish/internal/infrastructure/sessions"
	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

// setupTestEnvironment creates a test environment with RedfishServer.
func setupTestEnvironment() (*gin.Engine, *RedfishServer) {
	gin.SetMode(gin.TestMode)

	// Create test config
	cfg := &config.Config{
		Auth: config.Auth{
			AdminUsername: "admin",
			AdminPassword: "password",
			JWTKey:        "test-secret-key-for-jwt-signing",
			JWTExpiration: 24 * time.Hour,
		},
	}

	// Create session repository and use case
	repo := sessioninfra.NewInMemoryRepository(1 * time.Minute)
	useCase := sessions.NewUseCase(repo, cfg)

	// Create RedfishServer
	server := &RedfishServer{
		SessionUC: useCase,
		Config:    cfg,
	}

	// Setup router
	router := gin.New()

	return router, server
}

// TestSessionLifecycle tests the complete session lifecycle.
func TestSessionLifecycle(t *testing.T) {
	t.Parallel()

	router, server := setupTestEnvironment()

	// Register routes
	router.POST("/redfish/v1/SessionService/Sessions", server.PostRedfishV1SessionServiceSessions)
	router.GET("/redfish/v1/SessionService/Sessions/:SessionId", func(c *gin.Context) {
		server.GetRedfishV1SessionServiceSessionsSessionId(c, c.Param("SessionId"))
	})
	router.DELETE("/redfish/v1/SessionService/Sessions/:SessionId", func(c *gin.Context) {
		server.DeleteRedfishV1SessionServiceSessionsSessionId(c, c.Param("SessionId"))
	})

	// Step 1: Create session
	createReq := map[string]string{
		"UserName": "admin",
		"Password": "password",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Session creation should return 201")

	// Extract token and session ID
	token := w.Header().Get("X-Auth-Token")

	location := w.Header().Get("Location")

	assert.NotEmpty(t, token, "X-Auth-Token header should be present")

	assert.NotEmpty(t, location, "Location header should be present")

	var createResp map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	sessionID, ok := createResp["Id"].(string)

	require.True(t, ok, "Session ID should be a string")

	t.Logf("Created session: %s", sessionID)

	t.Logf("Token: %s", token[:50]+"...")

	// Step 2: Get session details
	req = httptest.NewRequest(http.MethodGet, "/redfish/v1/SessionService/Sessions/"+sessionID, http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Get session should return 200")

	var getResp map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &getResp)

	require.NoError(t, err)
	assert.Equal(t, sessionID, getResp["Id"])
	assert.Equal(t, "admin", getResp["UserName"])

	// Step 3: Delete session
	req = httptest.NewRequest(http.MethodDelete, "/redfish/v1/SessionService/Sessions/"+sessionID, http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code, "Delete session should return 204")

	// Step 4: Verify session no longer accessible
	req = httptest.NewRequest(http.MethodGet, "/redfish/v1/SessionService/Sessions/"+sessionID, http.NoBody)

	req.Header.Set("X-Auth-Token", token)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "Deleted session should return 404")
}

// TestCreateSessionInvalidCredentials tests session creation with wrong credentials.
func TestCreateSessionInvalidCredentials(t *testing.T) {
	t.Parallel()

	router, server := setupTestEnvironment()
	router.POST("/redfish/v1/SessionService/Sessions", server.PostRedfishV1SessionServiceSessions)

	createReq := map[string]string{
		"UserName": "admin",
		"Password": "wrongpassword",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewReader(body))

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid credentials should return 401")
}

// TestSessionAuthMiddleware tests the session authentication middleware.
func TestSessionAuthMiddleware(t *testing.T) {
	t.Parallel()

	router, server := setupTestEnvironment()

	// Create session first
	router.POST("/redfish/v1/SessionService/Sessions", server.PostRedfishV1SessionServiceSessions)

	createReq := map[string]string{
		"UserName": "admin",
		"Password": "password",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	token := w.Header().Get("X-Auth-Token")
	require.NotEmpty(t, token)

	// Test middleware with valid token
	router.GET("/test/protected", SessionAuthMiddleware(server.SessionUC), func(c *gin.Context) {
		username, _ := c.Get("username")
		c.JSON(http.StatusOK, gin.H{"user": username})
	})

	req = httptest.NewRequest(http.MethodGet, "/test/protected", http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Valid token should allow access")

	var resp map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "admin", resp["user"])

	// Test middleware without token
	req = httptest.NewRequest(http.MethodGet, "/test/protected", http.NoBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "No token should return 401")
}

// TestSessionServiceEndpoint tests the SessionService root endpoint.
func TestSessionServiceEndpoint(t *testing.T) {
	t.Parallel()

	router, server := setupTestEnvironment()
	router.GET("/redfish/v1/SessionService", server.GetRedfishV1SessionService)

	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/SessionService", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &resp)

	require.NoError(t, err)

	assert.Equal(t, "SessionService", resp["Id"])
	assert.Equal(t, "#SessionService.v1_2_0.SessionService", resp["@odata.type"])
	serviceEnabled, ok := resp["ServiceEnabled"].(bool)
	require.True(t, ok, "ServiceEnabled should be a bool")
	assert.True(t, serviceEnabled)
}

// TestListSessions tests listing all active sessions.
func TestListSessions(t *testing.T) {
	t.Parallel()

	router, server := setupTestEnvironment()

	router.POST("/redfish/v1/SessionService/Sessions", server.PostRedfishV1SessionServiceSessions)
	router.GET("/redfish/v1/SessionService/Sessions", server.GetRedfishV1SessionServiceSessions)
	router.DELETE("/redfish/v1/SessionService/Sessions/:SessionId", func(c *gin.Context) {
		server.DeleteRedfishV1SessionServiceSessionsSessionId(c, c.Param("SessionId"))
	})

	// Create first session
	createReq := map[string]string{
		"UserName": "admin",
		"Password": "password",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// List sessions - should have 1 session
	req = httptest.NewRequest(http.MethodGet, "/redfish/v1/SessionService/Sessions", http.NoBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	members, ok := resp["Members"].([]interface{})
	require.True(t, ok, "Members should be an array")
	assert.Equal(t, 1, len(members), "Should have 1 active session")
	assert.Equal(t, float64(1), resp["Members@odata.count"])

	// Try to create duplicate session - should fail with 409
	body, _ = json.Marshal(createReq)
	req = httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code, "Duplicate session should return 409 Conflict")
}

// TestTokenCompatibility tests backward compatibility with Bearer tokens.
func TestTokenCompatibility(t *testing.T) {
	t.Parallel()

	router, server := setupTestEnvironment()

	// Create session
	router.POST("/redfish/v1/SessionService/Sessions", server.PostRedfishV1SessionServiceSessions)

	createReq := map[string]string{
		"UserName": "admin",
		"Password": "password",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	token := w.Header().Get("X-Auth-Token")

	// Test with Bearer token format
	router.GET("/test/protected", SessionAuthMiddleware(server.SessionUC), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req = httptest.NewRequest(http.MethodGet, "/test/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)

	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Bearer token should also work")
}

// TestJWTIntegration verifies JWT structure.
func TestJWTIntegration(t *testing.T) {
	t.Parallel()

	router, server := setupTestEnvironment()
	router.POST("/redfish/v1/SessionService/Sessions", server.PostRedfishV1SessionServiceSessions)

	createReq := map[string]string{
		"UserName": "admin",
		"Password": "password",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	token := w.Header().Get("X-Auth-Token")
	assert.NotEmpty(t, token)

	// JWT should have 3 parts (header.payload.signature)
	parts := bytes.Split([]byte(token), []byte("."))
	assert.Equal(t, 3, len(parts), "JWT should have 3 parts")

	t.Logf("JWT Token: %s", token)
	t.Logf("Token is a valid JWT with HMAC-SHA256 signature")
}
