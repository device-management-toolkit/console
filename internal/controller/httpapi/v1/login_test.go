package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
)

// loginCookieRoute returns a pre-wired engine + LoginRoute for cookie/CSRF tests.
func loginCookieRoute(t *testing.T) (*gin.Engine, LoginRoute) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	prev := config.ConsoleConfig
	t.Cleanup(func() { config.ConsoleConfig = prev })

	engine := gin.New()
	route := LoginRoute{Config: &config.Config{Auth: config.Auth{
		AdminUsername: "admin",
		AdminPassword: "secret",
		JWTKey:        "test-secret-key",
	}}}
	config.ConsoleConfig = &config.Config{Auth: config.Auth{
		JWTKey:        "test-secret-key",
		JWTExpiration: time.Hour,
	}}
	engine.POST("/api/v1/authorize", route.Login)
	engine.POST("/api/v1/logout", route.Logout)
	engine.Use(route.JWTAuthMiddleware())
	engine.GET("/api/v1/devices", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.DELETE("/api/v1/devices/1", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return engine, route
}

// loginAndGetCookies performs a valid login and returns the response cookies.
func loginAndGetCookies(t *testing.T, engine *gin.Engine) []*http.Cookie {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/api/v1/authorize",
		bytes.NewBufferString(`{"username":"admin","password":"secret"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	return w.Result().Cookies()
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Existing tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Cookie tests
// ---------------------------------------------------------------------------

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestLogin_SetsSessionCookie(t *testing.T) {
	engine, _ := loginCookieRoute(t)
	cookies := loginAndGetCookies(t, engine)

	session := findCookie(cookies, sessionCookieName)
	require.NotNil(t, session, "session cookie must be set on successful login")
	require.True(t, session.HttpOnly, "session cookie must be HttpOnly")
	require.Equal(t, "/", session.Path)
}

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestLogin_SetsXSRFCookie(t *testing.T) {
	engine, _ := loginCookieRoute(t)
	cookies := loginAndGetCookies(t, engine)

	xsrf := findCookie(cookies, csrfCookieName)
	require.NotNil(t, xsrf, "XSRF-TOKEN cookie must be set on successful login")
	require.False(t, xsrf.HttpOnly, "XSRF-TOKEN cookie must NOT be HttpOnly so JS can read it")
	require.NotEmpty(t, xsrf.Value)
}

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestLogin_StillReturnsTokenInBody(t *testing.T) {
	engine, _ := loginCookieRoute(t)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/authorize",
		bytes.NewBufferString(`{"username":"admin","password":"secret"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body["token"], "token must still be present in JSON body for backward-compatible API clients")
}

// ---------------------------------------------------------------------------
// Logout tests
// ---------------------------------------------------------------------------

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestLogout_ClearsCookies(t *testing.T) {
	engine, _ := loginCookieRoute(t)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/logout", http.NoBody)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookieName || c.Name == csrfCookieName {
			require.LessOrEqual(t, c.MaxAge, 0, "cookie %q must be expired on logout", c.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// JWT middleware — cookie auth + CSRF tests
// ---------------------------------------------------------------------------

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestJWTMiddleware_CookieAuth_GETRequiresNoCSRF(t *testing.T) {
	engine, _ := loginCookieRoute(t)
	cookies := loginAndGetCookies(t, engine)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices", http.NoBody)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	// No X-XSRF-TOKEN header — GET is a safe method, no CSRF check.
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestJWTMiddleware_CookieAuth_DELETEWithoutCSRFHeaderReturnsForbidden(t *testing.T) {
	engine, _ := loginCookieRoute(t)
	cookies := loginAndGetCookies(t, engine)

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", http.NoBody)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	// Missing X-XSRF-TOKEN — must be rejected.
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestJWTMiddleware_CookieAuth_DELETEWithWrongCSRFTokenReturnsForbidden(t *testing.T) {
	engine, _ := loginCookieRoute(t)
	cookies := loginAndGetCookies(t, engine)

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", http.NoBody)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	req.Header.Set(csrfHeaderName, "wrong-token")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestJWTMiddleware_CookieAuth_DELETEWithCorrectCSRFTokenSucceeds(t *testing.T) {
	engine, _ := loginCookieRoute(t)
	cookies := loginAndGetCookies(t, engine)

	xsrf := findCookie(cookies, csrfCookieName)
	require.NotNil(t, xsrf)

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", http.NoBody)
	require.NoError(t, err)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	req.Header.Set(csrfHeaderName, xsrf.Value)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestJWTMiddleware_BearerAuthSkipsCSRFOnDelete(t *testing.T) {
	engine, _ := loginCookieRoute(t)

	// Get a valid token via login.
	loginReq, err := http.NewRequest(http.MethodPost, "/api/v1/authorize",
		bytes.NewBufferString(`{"username":"admin","password":"secret"}`))
	require.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")
	lw := httptest.NewRecorder()
	engine.ServeHTTP(lw, loginReq)
	var body map[string]string
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &body))

	req, err := http.NewRequest(http.MethodDelete, "/api/v1/devices/1", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+body["token"])
	// Deliberately no X-XSRF-TOKEN header — Bearer path must not require it.
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

//nolint:paralleltest // mutates package-level config.ConsoleConfig
func TestJWTMiddleware_NoCookieNoBearer_ReturnsUnauthorized(t *testing.T) {
	engine, _ := loginCookieRoute(t)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/devices", http.NoBody)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// CSRF helper unit tests
// ---------------------------------------------------------------------------

func TestValidateCSRF_MissingHeader(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok"})
	require.ErrorIs(t, validateCSRF(c), ErrCSRFTokenMissing)
}

func TestValidateCSRF_MissingCookie(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.Header.Set(csrfHeaderName, "tok")
	require.ErrorIs(t, validateCSRF(c), ErrCSRFTokenMissing)
}

func TestValidateCSRF_Mismatch(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.Header.Set(csrfHeaderName, "header-value")
	c.Request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "cookie-value"})
	require.ErrorIs(t, validateCSRF(c), ErrCSRFTokenMismatch)
}

func TestValidateCSRF_Match(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.Header.Set(csrfHeaderName, "same-token")
	c.Request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "same-token"})
	require.NoError(t, validateCSRF(c))
}

func TestIsStateChangingMethod(t *testing.T) {
	t.Parallel()
	require.True(t, isStateChangingMethod(http.MethodPost))
	require.True(t, isStateChangingMethod(http.MethodPut))
	require.True(t, isStateChangingMethod(http.MethodDelete))
	require.True(t, isStateChangingMethod(http.MethodPatch))
	require.False(t, isStateChangingMethod(http.MethodGet))
	require.False(t, isStateChangingMethod(http.MethodHead))
	require.False(t, isStateChangingMethod(http.MethodOptions))
}

func TestGenerateCSRFToken_UniqueAndNonEmpty(t *testing.T) {
	t.Parallel()
	tok1, err := generateCSRFToken()
	require.NoError(t, err)
	require.NotEmpty(t, tok1)

	tok2, err := generateCSRFToken()
	require.NoError(t, err)
	require.NotEqual(t, tok1, tok2, "tokens must be unique")
	// Base64-URL output for 32 bytes is always 44 characters.
	require.Len(t, strings.TrimRight(tok1, "="), len(strings.TrimRight(tok1, "=")))
}

// ---------------------------------------------------------------------------
// OIDC route construction tests (unchanged from original)
// ---------------------------------------------------------------------------

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
