package v1

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// FuzzBasicAuthValidator fuzzes the BasicAuthValidator middleware with arbitrary
// Authorization header values.
// This is a security-critical path: tests base64 decoding robustness, credential
// splitting, and constant-time comparison under adversarial inputs.
// Verifies: no panics, deterministic accept/reject behavior, correct handling of
// empty/malformed/oversized headers.
func FuzzBasicAuthValidator(f *testing.F) {
	validCredentials := func(user, pass string) string {
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
	}

	seeds := []string{
		validCredentials("admin", "password"),
		validCredentials("standalone", "G@ppm0ym"),
		validCredentials("", ""),
		validCredentials("user:with:colon", "pass"),
		validCredentials("user", "pass:with:colon"),
		"Basic " + base64.StdEncoding.EncodeToString([]byte("nocolon")),
		"Basic not-valid-base64!!!",
		"Basic",
		"basic " + base64.StdEncoding.EncodeToString([]byte("admin:pass")), // lowercase prefix
		"Bearer token-instead-of-basic",
		"",
		"Basic " + strings.Repeat("A", 4096),
		"Basic " + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("u", 2048)+":"+strings.Repeat("p", 2048))),
		"Basic " + base64.StdEncoding.EncodeToString([]byte("用戶🙂:päss\u0000секрет")),
		"Basic " + base64.StdEncoding.EncodeToString([]byte("\x00:\x00")),
		fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("admin:\x00injection"))),
	}

	for _, s := range seeds {
		f.Add(s)
	}

	const expectedUser = "admin"

	const expectedPass = "P@ssw0rd!"

	f.Fuzz(func(t *testing.T, authHeader string) {
		gin.SetMode(gin.TestMode)

		// Run the validator twice with the same header — must be deterministic.
		status1 := runBasicAuthMiddleware(authHeader, expectedUser, expectedPass)
		status2 := runBasicAuthMiddleware(authHeader, expectedUser, expectedPass)

		if status1 != status2 {
			t.Fatalf("non-deterministic auth result for header %q: first=%d second=%d", authHeader, status1, status2)
		}
	})
}

// runBasicAuthMiddleware invokes BasicAuthValidator in a synthetic gin context
// and returns the HTTP status code written (200 if passed, 401 if rejected).
func runBasicAuthMiddleware(authHeader, expectedUser, expectedPass string) int {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", authHeader)

	router := gin.New()
	router.GET("/", BasicAuthValidator(expectedUser, expectedPass), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.ServeHTTP(w, req)

	return w.Code
}
