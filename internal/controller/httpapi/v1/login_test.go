package v1

import (
	"bytes"
	"encoding/json"
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
	require.Equal(t, "Incorrect username or password", got["message"])
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
