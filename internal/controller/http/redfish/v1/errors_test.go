package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/config"
)

func TestRedfishJWTAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		config         *config.Config
		expectedStatus int
		checkResponse  func(t *testing.T, body string, headers http.Header)
	}{
		{
			name:       "missing authorization header",
			authHeader: "",
			config: &config.Config{
				Auth: config.Auth{
					Disabled: false,
					JWTKey:   "test-secret-key",
				},
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.Contains(t, body, `"Base.1.11.0.NoValidSession"`)
				assert.Contains(t, body, "no valid session established")
				assert.Equal(t, "application/json; charset=utf-8", headers.Get("Content-Type"))
				assert.Equal(t, "4.0", headers.Get("OData-Version"))
			},
		},
		{
			name:       "invalid JWT token",
			authHeader: "Bearer invalid.jwt.token",
			config: &config.Config{
				Auth: config.Auth{
					Disabled: false,
					JWTKey:   "test-secret-key",
				},
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.Contains(t, body, `"Base.1.11.0.NoValidSession"`)
				assert.Contains(t, body, "no valid session established")
			},
		},
		{
			name:       "expired JWT token",
			authHeader: createExpiredJWT("test-secret-key"),
			config: &config.Config{
				Auth: config.Auth{
					Disabled: false,
					JWTKey:   "test-secret-key",
				},
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.Contains(t, body, `"Base.1.11.0.NoValidSession"`)
			},
		},
		{
			name:       "valid JWT token",
			authHeader: createValidJWT("test-secret-key"),
			config: &config.Config{
				Auth: config.Auth{
					Disabled: false,
					JWTKey:   "test-secret-key",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.Equal(t, "success", body)
			},
		},
		{
			name:       "OAuth/OIDC config - not implemented",
			authHeader: "Bearer some.oauth.token",
			config: &config.Config{
				Auth: config.Auth{
					Disabled: false,
					ClientID: "oauth-client-id",
					JWTKey:   "test-secret-key",
				},
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				assert.Contains(t, body, `"Base.1.11.0.NoValidSession"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Gin
			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Add the middleware
			router.Use(RedfishJWTAuthMiddleware(tt.config))

			// Add a test endpoint
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "success")
			})

			// Create request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String(), w.Header())
			}
		})
	}
}

func TestSetRedfishHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SetRedfishHeaders(c)

	headers := w.Header()
	assert.Equal(t, "application/json; charset=utf-8", headers.Get("Content-Type"))
	assert.Equal(t, "4.0", headers.Get("OData-Version"))
	assert.Equal(t, "no-cache", headers.Get("Cache-Control"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
}

func TestRedfishErrorResponses(t *testing.T) {
	tests := []struct {
		name           string
		errorFunc      func(*gin.Context)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "MalformedJSONError",
			errorFunc:      MalformedJSONError,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Base.1.11.0.MalformedJSON",
		},
		{
			name: "PropertyMissingError",
			errorFunc: func(c *gin.Context) {
				PropertyMissingError(c, "TestProperty")
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Base.1.11.0.PropertyMissing",
		},
		{
			name: "PropertyValueNotInListError",
			errorFunc: func(c *gin.Context) {
				PropertyValueNotInListError(c, "InvalidValue", "TestProperty")
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Base.1.11.0.PropertyValueNotInList",
		},
		{
			name: "ResourceNotFoundError",
			errorFunc: func(c *gin.Context) {
				ResourceNotFoundError(c, "ComputerSystem", "test-id")
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "Base.1.11.0.ResourceNotFound",
		},
		{
			name:           "OperationNotAllowedError",
			errorFunc:      OperationNotAllowedError,
			expectedStatus: http.StatusConflict,
			expectedMsg:    "Base.1.11.0.OperationNotAllowed",
		},
		{
			name: "MethodNotAllowedError",
			errorFunc: func(c *gin.Context) {
				MethodNotAllowedError(c, "TestAction", "POST")
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedMsg:    "Base.1.11.0.ActionNotSupported",
		},
		{
			name:           "NoValidSessionError",
			errorFunc:      NoValidSessionError,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "Base.1.11.0.NoValidSession",
		},
		{
			name:           "InsufficientPrivilegeError",
			errorFunc:      InsufficientPrivilegeError,
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "Base.1.11.0.InsufficientPrivilege",
		},
		{
			name:           "GeneralError",
			errorFunc:      GeneralError,
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Base.1.11.0.GeneralError",
		},
		{
			name:           "ServiceUnavailableError",
			errorFunc:      ServiceUnavailableError,
			expectedStatus: http.StatusBadGateway,
			expectedMsg:    "Base.1.11.0.GeneralError",
		},
		{
			name:           "ServiceTemporarilyUnavailableError",
			errorFunc:      ServiceTemporarilyUnavailableError,
			expectedStatus: http.StatusServiceUnavailable,
			expectedMsg:    "Base.1.11.0.GeneralError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/test", nil)
			c.Request = req

			// Call the error function
			tt.errorFunc(c)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			body := w.Body.String()
			assert.Contains(t, body, tt.expectedMsg)
			assert.Contains(t, body, `"@Message.ExtendedInfo"`)
			assert.Contains(t, body, `"error"`)

			// Check headers
			headers := w.Header()
			assert.Equal(t, "application/json; charset=utf-8", headers.Get("Content-Type"))
			assert.Equal(t, "4.0", headers.Get("OData-Version"))

			// For MethodNotAllowedError, check Allow header
			if tt.name == "MethodNotAllowedError" {
				assert.Equal(t, "POST", headers.Get("Allow"))
			}

			// For ServiceTemporarilyUnavailableError, check Retry-After header
			if tt.name == "ServiceTemporarilyUnavailableError" {
				assert.Equal(t, "30", headers.Get("Retry-After"))
			}
		})
	}
}

// Helper functions for JWT token creation

func createValidJWT(secretKey string) string {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secretKey))

	return "Bearer " + tokenString
}

func createExpiredJWT(secretKey string) string {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired 1 hour ago
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secretKey))

	return "Bearer " + tokenString
}
