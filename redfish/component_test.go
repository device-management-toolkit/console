// Package redfish provides unit tests for the Redfish component initialization and routing.
package redfish

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	v1 "github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/handler"
	sessioninfra "github.com/device-management-toolkit/console/redfish/internal/infrastructure/sessions"
	"github.com/device-management-toolkit/console/redfish/internal/mocks"
	redfishusecase "github.com/device-management-toolkit/console/redfish/internal/usecase"
	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

// setupTestServer creates a test environment with the Redfish server configured.
// Note: This function modifies global state (server, componentConfig) so tests using it cannot run in parallel.
func setupTestServer(t *testing.T) (*gin.Engine, *v1.RedfishServer) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Auth: config.Auth{
			AdminUsername: "admin",
			AdminPassword: "testpassword",
			JWTKey:        "test-jwt-key-for-component-tests",
			JWTExpiration: 1 * time.Hour,
			Disabled:      false,
		},
	}

	// Create mock repository for computer systems
	mockRepo := mocks.NewMockComputerSystemRepo()

	computerSystemUC := &redfishusecase.ComputerSystemUseCase{Repo: mockRepo}

	// Create session repository and use case
	const sessionCleanupInterval = 1 * time.Minute

	sessionRepo := sessioninfra.NewInMemoryRepository(sessionCleanupInterval)
	sessionUC := sessions.NewUseCase(sessionRepo, cfg)

	// Create the server
	testServer := &v1.RedfishServer{
		ComputerSystemUC: computerSystemUC,
		SessionUC:        sessionUC,
		Config:           cfg,
	}

	// Load services
	services, err := v1.ExtractServicesFromOpenAPIData(embeddedOpenAPISpec)
	if err != nil {
		testServer.Services = v1.GetDefaultServices()
	} else {
		testServer.Services = services
	}

	// Set global server and config for middleware
	server = testServer
	componentConfig = &ComponentConfig{
		Enabled:      true,
		AuthRequired: true,
		BaseURL:      "/redfish/v1",
	}

	router := gin.New()

	return router, testServer
}

// TestIsPublicEndpoint tests the public endpoint detection logic.
func TestIsPublicEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		method   string
		expected bool
	}{
		{
			name:     "Service root is public",
			path:     "/redfish/v1/",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Metadata is public",
			path:     "/redfish/v1/$metadata",
			method:   "GET",
			expected: true,
		},
		{
			name:     "OData service document is public",
			path:     "/redfish/v1/odata",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Session creation POST is public",
			path:     "/redfish/v1/SessionService/Sessions",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Session collection GET is protected",
			path:     "/redfish/v1/SessionService/Sessions",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Systems collection is protected",
			path:     "/redfish/v1/Systems",
			method:   "GET",
			expected: false,
		},
		{
			name:     "SessionService is protected",
			path:     "/redfish/v1/SessionService",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Individual session is protected",
			path:     "/redfish/v1/SessionService/Sessions/12345",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Session DELETE is protected",
			path:     "/redfish/v1/SessionService/Sessions/12345",
			method:   "DELETE",
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isPublicEndpoint(tt.path, tt.method)
			assert.Equal(t, tt.expected, result, "isPublicEndpoint(%q, %q)", tt.path, tt.method)
		})
	}
}

// TestComponentConfig tests the component configuration defaults.
func TestComponentConfig(t *testing.T) {
	t.Parallel()

	cfg := &ComponentConfig{
		Enabled:      true,
		AuthRequired: true,
		BaseURL:      "/redfish/v1",
	}

	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.AuthRequired)
	assert.Equal(t, "/redfish/v1", cfg.BaseURL)
}

// TestCreateAuthMiddleware_SessionCreationPublic tests that session creation is accessible without auth.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestCreateAuthMiddleware_SessionCreationPublic(t *testing.T) {
	router, testServer := setupTestServer(t)

	// Register session creation route with middleware (convert to gin.HandlerFunc)
	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.POST("/redfish/v1/SessionService/Sessions", middlewares, testServer.PostRedfishV1SessionServiceSessions)

	// Create session without authentication - should succeed
	req := httptest.NewRequest(http.MethodPost, "/redfish/v1/SessionService/Sessions", http.NoBody)
	req.Header.Set("Content-Type", "application/json")

	// The middleware should allow this through
	// Just test that middleware doesn't block - actual handler may fail due to missing body
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not get 401 Unauthorized - may get 400 for bad request body
	assert.NotEqual(t, http.StatusUnauthorized, w.Code, "Session creation should not require auth")
}

// TestCreateAuthMiddleware_ProtectedEndpoint tests that protected endpoints require auth.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestCreateAuthMiddleware_ProtectedEndpoint(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Access protected endpoint without authentication
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Protected endpoint should require auth")
}

// TestCreateAuthMiddleware_PublicEndpoint tests that public endpoints don't require auth.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestCreateAuthMiddleware_PublicEndpoint(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/", middlewares, testServer.GetRedfishV1)

	// Access service root without authentication
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 200 OK (public endpoint)
	assert.Equal(t, http.StatusOK, w.Code, "Service root should be publicly accessible")
}

// TestCreateErrorHandler tests the error handler function.
func TestCreateErrorHandler(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		statusCode     int
		err            error
		expectedStatus int
	}{
		{
			name:           "Unauthorized error",
			statusCode:     statusUnauthorized,
			err:            nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Forbidden error",
			statusCode:     statusForbidden,
			err:            nil,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Method not allowed error",
			statusCode:     statusMethodNotAllowed,
			err:            nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Bad request error",
			statusCode:     statusBadRequest,
			err:            assert.AnError,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			errorHandler := createErrorHandler()

			router.GET("/test", func(c *gin.Context) {
				errorHandler(c, tt.err, tt.statusCode)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestErrDevicesCastFailed tests the error variable.
func TestErrDevicesCastFailed(t *testing.T) {
	t.Parallel()

	require.NotNil(t, ErrDevicesCastFailed)
	assert.Equal(t, "failed to cast devices use case", ErrDevicesCastFailed.Error())
}

// TestComponentConfigDefaults verifies the component config can be properly initialized.
func TestComponentConfigDefaults(t *testing.T) {
	t.Parallel()

	// Test with auth enabled
	cfg := &ComponentConfig{
		Enabled:      true,
		AuthRequired: true,
		BaseURL:      "/redfish/v1",
	}

	assert.True(t, cfg.Enabled, "Component should be enabled")
	assert.True(t, cfg.AuthRequired, "Auth should be required")
	assert.Equal(t, "/redfish/v1", cfg.BaseURL)

	// Test with auth disabled
	cfgNoAuth := &ComponentConfig{
		Enabled:      true,
		AuthRequired: false,
		BaseURL:      "/redfish/v1",
	}

	assert.True(t, cfgNoAuth.Enabled)
	assert.False(t, cfgNoAuth.AuthRequired, "Auth should not be required")
}

// TestEmbeddedOpenAPISpec tests that the OpenAPI spec is properly embedded.
func TestEmbeddedOpenAPISpec(t *testing.T) {
	t.Parallel()

	require.NotNil(t, embeddedOpenAPISpec, "OpenAPI spec should be embedded")
	assert.Greater(t, len(embeddedOpenAPISpec), 0, "OpenAPI spec should not be empty")
}

// TestExtractServicesFromOpenAPIData tests service extraction from embedded spec.
func TestExtractServicesFromOpenAPIData(t *testing.T) {
	t.Parallel()

	services, err := v1.ExtractServicesFromOpenAPIData(embeddedOpenAPISpec)

	// Should either succeed or we fall back to defaults
	if err != nil {
		// Verify we can get defaults
		defaultServices := v1.GetDefaultServices()
		require.NotNil(t, defaultServices)
		assert.Greater(t, len(defaultServices), 0)
	} else {
		require.NotNil(t, services)
		assert.Greater(t, len(services), 0)
	}
}

// TestXAuthToken_ValidToken tests that a valid X-Auth-Token grants access to protected endpoints.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_ValidToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// First, create a session to get a valid token
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)
	require.NotEmpty(t, token)

	// Access protected endpoint with valid X-Auth-Token
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 200 OK with valid token
	assert.Equal(t, http.StatusOK, w.Code, "Valid X-Auth-Token should grant access")
}

// TestXAuthToken_InvalidToken tests that an invalid X-Auth-Token is rejected.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_InvalidToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Access protected endpoint with invalid X-Auth-Token
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
	req.Header.Set("X-Auth-Token", "invalid-token-12345")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 401 Unauthorized with invalid token
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid X-Auth-Token should be rejected")
}

// TestXAuthToken_EmptyToken tests that an empty X-Auth-Token falls back to Basic Auth.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_EmptyToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Access protected endpoint with empty X-Auth-Token (should fall back to Basic Auth)
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
	req.Header.Set("X-Auth-Token", "")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 401 because no Basic Auth provided either
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Empty X-Auth-Token without Basic Auth should be rejected")
}

// TestXAuthToken_TakesPrecedenceOverBasicAuth tests that X-Auth-Token takes precedence over Basic Auth.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_TakesPrecedenceOverBasicAuth(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Create a valid session token
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)
	require.NotEmpty(t, token)

	// Access with both X-Auth-Token and Basic Auth (wrong credentials)
	// X-Auth-Token should take precedence and succeed
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
	req.Header.Set("X-Auth-Token", token)
	req.SetBasicAuth("wronguser", "wrongpassword")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed because X-Auth-Token is valid (takes precedence)
	assert.Equal(t, http.StatusOK, w.Code, "Valid X-Auth-Token should take precedence over invalid Basic Auth")
}

// TestXAuthToken_InvalidTokenWithValidBasicAuth tests behavior with invalid token but valid Basic Auth.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_InvalidTokenWithValidBasicAuth(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Access with invalid X-Auth-Token and valid Basic Auth
	// Per Redfish spec, X-Auth-Token takes precedence, so this should fail
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
	req.Header.Set("X-Auth-Token", "invalid-token")
	req.SetBasicAuth("admin", "testpassword")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail because X-Auth-Token is checked first and is invalid
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid X-Auth-Token should fail even with valid Basic Auth")
}

// TestXAuthToken_ExpiredToken tests that an expired session token is rejected.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_ExpiredToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Create a session and then delete it to simulate expiration
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)

	// Delete the session (simulating logout/expiration)
	err = testServer.SessionUC.DeleteSession(session.ID)
	require.NoError(t, err)

	// Try to use the token after session deletion
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 401 because session no longer exists
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Deleted session token should be rejected")
}

// TestXAuthToken_SessionServiceWithToken tests accessing SessionService with X-Auth-Token.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_SessionServiceWithToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/SessionService", middlewares, testServer.GetRedfishV1SessionService)

	// Create a session to get a valid token
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)

	// Access SessionService with valid token
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/SessionService", http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "SessionService should be accessible with valid X-Auth-Token")
}

// TestXAuthToken_SessionCollectionWithToken tests listing sessions with X-Auth-Token.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_SessionCollectionWithToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/SessionService/Sessions", middlewares, testServer.GetRedfishV1SessionServiceSessions)

	// Create a session to get a valid token
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)

	// Access session collection with valid token
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/SessionService/Sessions", http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Session collection should be accessible with valid X-Auth-Token")
}

// TestXAuthToken_IndividualSessionWithToken tests getting an individual session with X-Auth-Token.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_IndividualSessionWithToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/SessionService/Sessions/:SessionId", middlewares, func(c *gin.Context) {
		testServer.GetRedfishV1SessionServiceSessionsSessionId(c, c.Param("SessionId"))
	})

	// Create a session to get a valid token
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)

	// Access individual session with valid token
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/SessionService/Sessions/"+session.ID, http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Individual session should be accessible with valid X-Auth-Token")
}

// TestXAuthToken_DeleteSessionWithToken tests deleting a session using X-Auth-Token.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_DeleteSessionWithToken(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.DELETE("/redfish/v1/SessionService/Sessions/:SessionId", middlewares, func(c *gin.Context) {
		testServer.DeleteRedfishV1SessionServiceSessionsSessionId(c, c.Param("SessionId"))
	})

	// Create a session to get a valid token
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)

	// Delete session with valid token
	req := httptest.NewRequest(http.MethodDelete, "/redfish/v1/SessionService/Sessions/"+session.ID, http.NoBody)
	req.Header.Set("X-Auth-Token", token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code, "Session deletion should work with valid X-Auth-Token")
}

// TestXAuthToken_NoTokenNoBasicAuth tests that protected endpoints reject requests without any auth.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_NoTokenNoBasicAuth(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Access protected endpoint without any authentication
	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "No authentication should be rejected")
}

// TestXAuthToken_MalformedJWT tests that a malformed JWT token is rejected.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_MalformedJWT(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)

	// Try various malformed tokens
	malformedTokens := []string{
		"not-a-jwt",
		"header.payload",                 // Missing signature
		"header.payload.signature.extra", // Too many parts
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid", // Invalid payload
		"",           // Empty string
		"   ",        // Whitespace
		"Bearer xyz", // Wrong format (should be raw token)
	}

	for _, malformedToken := range malformedTokens {
		if malformedToken == "" {
			continue // Empty token falls back to Basic Auth
		}

		req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems", http.NoBody)
		req.Header.Set("X-Auth-Token", malformedToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "Malformed token %q should be rejected", malformedToken)
	}
}

// TestXAuthToken_MultipleEndpoints tests X-Auth-Token works across multiple protected endpoints.
//
//nolint:paralleltest // Cannot run in parallel - modifies global state (server, componentConfig)
func TestXAuthToken_MultipleEndpoints(t *testing.T) {
	router, testServer := setupTestServer(t)

	middlewares := gin.HandlerFunc(createAuthMiddleware())

	// Register multiple protected endpoints
	router.GET("/redfish/v1/Systems", middlewares, testServer.GetRedfishV1Systems)
	router.GET("/redfish/v1/SessionService", middlewares, testServer.GetRedfishV1SessionService)
	router.GET("/redfish/v1/SessionService/Sessions", middlewares, testServer.GetRedfishV1SessionServiceSessions)

	// Create a session to get a valid token
	session, token, err := testServer.SessionUC.CreateSession("admin", "testpassword", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, session)

	// Test multiple endpoints with the same token
	endpoints := []string{
		"/redfish/v1/Systems",
		"/redfish/v1/SessionService",
		"/redfish/v1/SessionService/Sessions",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
		req.Header.Set("X-Auth-Token", token)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Endpoint %s should be accessible with valid X-Auth-Token", endpoint)
	}
}
