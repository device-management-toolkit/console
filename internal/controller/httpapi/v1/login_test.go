package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/device-management-toolkit/console/config"
)

const (
	testJWTKey       = "test-jwt-key"
	testAdminUser    = "admin"
	testAdminPass    = "secret"
	testCredsBody    = `{"username":"admin","password":"secret"}`
	testProtectedURL = "/api/v1/devices"
	testAuthorizeURL = "/api/v1/authorize"
	testLogoutURL    = "/api/v1/authorize/logout"
)

// cookieAuthTestConfig is a basic-auth (non-OIDC) config with cookies enabled.
func cookieAuthTestConfig(t *testing.T) *config.Config {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(testAdminPass), bcrypt.DefaultCost)
	require.NoError(t, err)

	cfg := &config.Config{}
	cfg.AdminUsername = testAdminUser
	cfg.AdminPassword = string(hash)
	cfg.JWTKey = testJWTKey
	cfg.JWTExpiration = time.Hour
	cfg.CookieEnabled = true
	cfg.CookieName = config.DefaultSessionCookieName
	cfg.CookieSecure = true
	cfg.CookieSameSite = "strict"

	return cfg
}

// newAuthTestEngine wires authorize, logout and one protected route against cfg,
// installing it as the singleton the handlers read.
func newAuthTestEngine(t *testing.T, cfg *config.Config) *gin.Engine {
	t.Helper()

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	route := LoginRoute{Config: cfg}

	engine := gin.New()
	engine.POST(testAuthorizeURL, route.Login)
	engine.POST(testLogoutURL, route.Logout)
	engine.GET(testProtectedURL, route.JWTAuthMiddleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

	return engine
}

// login authorizes and returns the body token plus the cookies set, by name.
func login(t *testing.T, engine *gin.Engine) (token string, cookies map[string]*http.Cookie) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, testAuthorizeURL, bytes.NewBufferString(testCredsBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]string

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	cookies = make(map[string]*http.Cookie)
	for _, cookie := range w.Result().Cookies() {
		cookies[cookie.Name] = cookie
	}

	return body["token"], cookies
}

// get calls the protected route after applying mutators.
func get(t *testing.T, engine *gin.Engine, mutators ...func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, testProtectedURL, http.NoBody)
	require.NoError(t, err)

	for _, mutate := range mutators {
		mutate(req)
	}

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	return w
}

func withBearer(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

func withCookie(cookie *http.Cookie) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(cookie) }
}

// TestAuthorizeIssuesSessionCookies pins the shape REST clients rely on: the
// token stays in the body and the cookies are purely additive.
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestAuthorizeIssuesSessionCookies(t *testing.T) {
	engine := newAuthTestEngine(t, cookieAuthTestConfig(t))

	token, cookies := login(t, engine)
	require.NotEmpty(t, token, "token must remain in the response body for bearer clients")

	session, ok := cookies[config.DefaultSessionCookieName]
	require.True(t, ok, "expected the session cookie to be set")
	require.Equal(t, token, session.Value, "session cookie must carry the same token as the body")
	require.True(t, session.HttpOnly, "session cookie must not be readable by script")
	require.True(t, session.Secure)
	require.Equal(t, http.SameSiteStrictMode, session.SameSite)
	require.Equal(t, "/", session.Path)
	require.Positive(t, session.MaxAge)

	require.Len(t, cookies, 1, "only the session cookie is issued")
}

// TestBearerAuthUnchanged is the backward-compatibility guarantee for curl,
// Postman and partner tooling: a bearer header authenticates on its own.
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestBearerAuthUnchanged(t *testing.T) {
	engine := newAuthTestEngine(t, cookieAuthTestConfig(t))

	token, cookies := login(t, engine)

	t.Run("bearer header alone is accepted", func(t *testing.T) {
		require.Equal(t, http.StatusOK, get(t, engine, withBearer(token)).Code)
	})

	t.Run("bearer header is accepted alongside a cookie", func(t *testing.T) {
		session := cookies[config.DefaultSessionCookieName]

		w := get(t, engine, withBearer(token), withCookie(session))
		require.Equal(t, http.StatusOK, w.Code, "header takes precedence over the cookie")
	})

	t.Run("no credentials at all is still 401", func(t *testing.T) {
		require.Equal(t, http.StatusUnauthorized, get(t, engine).Code)
	})
}

// TestCookieAuthAcceptsSessionCookie covers the cookie path on its own.
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestCookieAuthAcceptsSessionCookie(t *testing.T) {
	engine := newAuthTestEngine(t, cookieAuthTestConfig(t))

	_, cookies := login(t, engine)
	session := cookies[config.DefaultSessionCookieName]

	t.Run("session cookie alone is accepted", func(t *testing.T) {
		require.Equal(t, http.StatusOK, get(t, engine, withCookie(session)).Code)
	})

	t.Run("a tampered session cookie is rejected", func(t *testing.T) {
		forged := &http.Cookie{Name: session.Name, Value: session.Value + "x"}
		require.Equal(t, http.StatusUnauthorized, get(t, engine, withCookie(forged)).Code)
	})
}

// TestCookieAuthDisabled checks the escape hatch: with cookies off, none are
// issued and an earlier one no longer authenticates.
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestCookieAuthDisabled(t *testing.T) {
	enabled := newAuthTestEngine(t, cookieAuthTestConfig(t))
	_, cookies := login(t, enabled)
	session := cookies[config.DefaultSessionCookieName]

	cfg := cookieAuthTestConfig(t)
	cfg.CookieEnabled = false
	disabled := newAuthTestEngine(t, cfg)

	token, issued := login(t, disabled)
	require.NotEmpty(t, token, "the body token is the only credential when cookies are off")
	require.Empty(t, issued, "no cookies should be issued when cookie auth is disabled")

	w := get(t, disabled, withCookie(session))
	require.Equal(t, http.StatusUnauthorized, w.Code, "a previously issued cookie must not authenticate")

	require.Equal(t, http.StatusOK, get(t, disabled, withBearer(token)).Code)
}

// TestLogoutWithoutConfig: the public logout route must not panic when the
// config singleton has not been populated yet.
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestLogoutWithoutConfig(t *testing.T) {
	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = nil

	engine := gin.New()
	route := LoginRoute{Config: &config.Config{}}
	engine.POST(testLogoutURL, route.Logout)

	req, err := http.NewRequest(http.MethodPost, testLogoutURL, http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()

	require.NotPanics(t, func() { engine.ServeHTTP(w, req) })
	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Result().Cookies(), "no cookie can be written without config")
}

// TestLogoutExpiresSessionCookie checks the cookie is cleared and that logout
// works without credentials.
//
//nolint:paralleltest // shared global config.ConsoleConfig
func TestLogoutExpiresSessionCookie(t *testing.T) {
	engine := newAuthTestEngine(t, cookieAuthTestConfig(t))

	req, err := http.NewRequest(http.MethodPost, testLogoutURL, http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "logout must work without a valid session")

	cleared := make(map[string]*http.Cookie)
	for _, cookie := range w.Result().Cookies() {
		cleared[cookie.Name] = cookie
	}

	cookie, ok := cleared[config.DefaultSessionCookieName]
	require.True(t, ok, "expected the session cookie to be expired")
	require.Empty(t, cookie.Value)
	require.Negative(t, cookie.MaxAge)
}

func TestLogin_InvalidCredentialsReturnsMessage(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	engine := gin.New()
	route := LoginRoute{Config: &config.Config{Auth: config.Auth{AdminUsername: "admin", AdminPassword: string(hash)}}}
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
