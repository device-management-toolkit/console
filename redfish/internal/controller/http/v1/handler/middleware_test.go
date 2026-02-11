package v1

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestBasicAuthValidator tests the BasicAuthValidator middleware.
func TestBasicAuthValidator(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	expectedUsername := "testuser"
	expectedPassword := "testpass"

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid credentials",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass")),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid credentials",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("wronguser:wrongpass")),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid auth format - Bearer instead of Basic",
			authHeader:     "Bearer token123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid base64 encoding",
			authHeader:     "Basic invalid!!!base64",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing colon separator in credentials",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("usernameonly")),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			router.GET("/test", BasicAuthValidator(expectedUsername, expectedPassword), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
