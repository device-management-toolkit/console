package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
)

func TestLogin_InvalidCredentialsReturnsMessage(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	route := LoginRoute{Config: &config.Config{Auth: config.Auth{AdminUsername: "admin", AdminPassword: "secret"}}}
	engine.POST("/api/v1/authorize", route.Login)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/authorize", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)

	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "invalid credentials", got["error"])
	require.Equal(t, "Incorrect Username and/or Password!", got["message"])
}

func TestLogin_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	route := LoginRoute{Config: &config.Config{Auth: config.Auth{AdminUsername: "admin", AdminPassword: "secret"}}}
	engine.POST("/api/v1/authorize", route.Login)

	makeRequest := func() *httptest.ResponseRecorder {
		req, err := http.NewRequest(http.MethodPost, "/api/v1/authorize", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// All requests from the same IP so they share one limiter bucket.
		req.RemoteAddr = "192.0.2.1:1234"

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		return w
	}

	// Exhaust the burst allowance (loginRateBurst == 5).
	for range loginRateBurst {
		w := makeRequest()
		require.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// The next request must be throttled.
	w := makeRequest()
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	require.Equal(t, "60", w.Header().Get("Retry-After"))

	var got map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "too many requests", got["error"])
}

// TestLogin_SpoofedXForwardedForCannotBypassRateLimit verifies that when no
// proxies are trusted (the default), rotating the X-Forwarded-For header does
// not create a fresh limiter bucket per request. Gin's default engine trusts
// all proxies, so we explicitly clear the trusted proxy list to mirror the
// production wiring in app.go (SetTrustedProxies(nil)).
func TestLogin_SpoofedXForwardedForCannotBypassRateLimit(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	// Mirror production: trust no proxies, so c.ClientIP() uses RemoteAddr and
	// ignores any client-supplied X-Forwarded-For header.
	require.NoError(t, engine.SetTrustedProxies(nil))

	route := LoginRoute{Config: &config.Config{Auth: config.Auth{AdminUsername: "admin", AdminPassword: "secret"}}}
	engine.POST("/api/v1/authorize", route.Login)

	makeRequest := func(spoofedIP string) *httptest.ResponseRecorder {
		req, err := http.NewRequest(http.MethodPost, "/api/v1/authorize", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// Same real connection IP, but a different forged X-Forwarded-For each time.
		req.RemoteAddr = "192.0.2.50:1234"
		req.Header.Set("X-Forwarded-For", spoofedIP)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		return w
	}

	// Exhaust the burst using a different spoofed IP on every request. If the
	// header were trusted, each would get its own bucket and never throttle.
	for i := range loginRateBurst {
		w := makeRequest(fmt.Sprintf("10.10.10.%d", i))
		require.Equal(t, http.StatusUnauthorized, w.Code)
	}

	// Despite a fresh spoofed X-Forwarded-For, the shared RemoteAddr bucket is
	// now empty, so this request must be throttled.
	w := makeRequest("10.10.10.99")
	require.Equal(t, http.StatusTooManyRequests, w.Code)
}

// oidcDiscoveryServer spins up a TLS test server that serves the minimum
// OpenID Connect discovery document that go-oidc requires. The issuer field
// in the response must match the server URL, otherwise NewProvider fails.
func oidcDiscoveryServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	srv := httptest.NewTLSServer(mux)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                srv.URL,
			"authorization_endpoint":                srv.URL + "/authorize",
			"token_endpoint":                        srv.URL + "/token",
			"jwks_uri":                              srv.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	t.Cleanup(srv.Close)

	return srv
}

// TestNewLoginRoute mutates the package-level config.ConsoleConfig, so its
// subtests run sequentially to avoid racing the global with other tests in
// this package that read it.
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestNewLoginRoute(t *testing.T) {
	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	t.Run("no ClientID returns route with nil Verifier", func(t *testing.T) {
		config.ConsoleConfig = &config.Config{}

		lr := NewLoginRoute(&config.Config{})

		require.NotNil(t, lr)
		require.Nil(t, lr.Verifier)
	})

	t.Run("TLSSkipVerify trusts self-signed IdP", func(t *testing.T) {
		srv := oidcDiscoveryServer(t)

		config.ConsoleConfig = &config.Config{}
		config.ConsoleConfig.ClientID = "test-client"
		config.ConsoleConfig.Issuer = srv.URL
		config.ConsoleConfig.TLSSkipVerify = true

		lr := NewLoginRoute(&config.Config{})

		require.NotNil(t, lr, "expected provider discovery to succeed with TLSSkipVerify=true")
		require.NotNil(t, lr.Verifier)
	})

	t.Run("default TLS verify rejects self-signed IdP", func(t *testing.T) {
		srv := oidcDiscoveryServer(t)

		config.ConsoleConfig = &config.Config{}
		config.ConsoleConfig.ClientID = "test-client"
		config.ConsoleConfig.Issuer = srv.URL
		config.ConsoleConfig.TLSSkipVerify = false

		lr := NewLoginRoute(&config.Config{})

		require.Nil(t, lr, "expected provider discovery to fail against self-signed cert without skip verify")
	})
}
