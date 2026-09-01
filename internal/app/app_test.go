package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestSecurityHeadersSetsNoSniff(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.Use(securityHeaders())
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"a": 1}) })
	// SPA fallback, the path the finding was reported against.
	r.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html></html>"))
	})
	r.GET("/unauth", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	})

	for _, path := range []string{"/ok", "/no/such/route", "/unauth"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))

		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), path)
	}
}

// The nosniff middleware has to sit ahead of CORS in the engine chain: the CORS
// middleware answers preflights and rejects disallowed origins itself, so
// anything registered after it never runs on those responses.
//
//nolint:paralleltest // setupHTTPHandler mutates global gin mode and reads GIN_MODE, which TestRun writes
func TestSetupHTTPHandlerSetsNoSniffAheadOfCORS(t *testing.T) {
	cfg := &config.Config{}
	cfg.AllowedOrigins = []string{"https://allowed.example"}
	cfg.AllowedHeaders = []string{"Content-Type"}
	cfg.Disabled = true

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	handler := setupHTTPHandler(cfg, logger.New("error"), &usecase.Usecases{})

	tests := []struct {
		name      string
		method    string
		origin    string
		preflight bool
		wantCode  int
	}{
		{"CORS rejects a disallowed preflight", http.MethodOptions, "https://blocked.example", true, http.StatusForbidden},
		{"CORS rejects a disallowed request", http.MethodGet, "https://blocked.example", false, http.StatusForbidden},
		{"allowed origin reaches the route", http.MethodGet, "https://allowed.example", false, http.StatusOK},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, "/healthz", http.NoBody)
		req.Header.Set("Origin", tc.origin)

		if tc.preflight {
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		}

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		require.Equal(t, tc.wantCode, w.Code, tc.name)
		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), tc.name)
	}
}

func TestNewOriginChecker(t *testing.T) {
	t.Parallel()

	allowed := []string{"https://allowed.example"}

	tests := []struct {
		name string
		// allow defaults to `allowed` when nil.
		allow []string
		// host overrides the request Host, i.e. what counts as same-origin.
		host   string
		origin string
		want   bool
	}{
		{name: "explicit origin is allowed", origin: "https://allowed.example", want: true},
		{name: "different origin is rejected", origin: "https://evil.example", want: false},
		{name: "missing origin is allowed for non-browser clients", origin: "", want: true},

		// A wildcard must not reopen the relay to arbitrary sites; it degrades
		// to same-origin only. Covers the shipped `allowed_origins: ["*"]` that
		// pre-existing installs still carry on disk.
		{name: "wildcard rejects a cross-origin page", allow: []string{"*"}, origin: "https://evil.example", want: false},
		{name: "wildcard still allows same-origin", allow: []string{"*"}, host: "console.example:8181", origin: "https://console.example:8181", want: true},
		{name: "empty allowlist rejects a cross-origin page", allow: []string{}, origin: "https://evil.example", want: false},
		{name: "empty allowlist still allows same-origin", allow: []string{}, host: "console.example:8181", origin: "https://console.example:8181", want: true},

		// Opaque origins: a sandboxed iframe sends the literal "null", and
		// data:/file: pages send no usable scheme://host pair either.
		{name: "null origin is rejected", origin: "null", want: false},
		{name: "opaque non-URL origin is rejected", origin: "evil.example", want: false},
		{name: "scheme-only origin is rejected", origin: "https://", want: false},

		// url.Parse lowercases the scheme but not the host, while
		// gin-contrib/cors lowercases both. A mixed-case allowlist entry must
		// not pass CORS and then fail the relay.
		{name: "mixed-case allowlist entry matches lowercase origin", allow: []string{"https://Allowed.Example"}, origin: "https://allowed.example", want: true},
		{name: "mixed-case origin matches lowercase allowlist entry", origin: "https://ALLOWED.example", want: true},
		{name: "mixed-case same-origin host matches", host: "Console.Example:8181", origin: "https://console.example:8181", want: true},

		// The Origin header carries no path, but a trailing slash in config is
		// an easy mistake to make.
		{name: "trailing slash in allowlist entry is tolerated", allow: []string{"https://allowed.example/"}, origin: "https://allowed.example", want: true},

		// Scheme and port are part of the origin: neither may be substituted.
		{name: "scheme mismatch is rejected", origin: "http://allowed.example", want: false},
		{name: "port mismatch is rejected", allow: []string{"https://allowed.example:8181"}, origin: "https://allowed.example:4200", want: false},
		{name: "suffix of an allowed host is rejected", origin: "https://evil-allowed.example", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			allow := tc.allow
			if allow == nil {
				allow = allowed
			}

			checker := newOriginChecker(allow)
			req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx", http.NoBody)

			if tc.host != "" {
				req.Host = tc.host
			}

			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			require.Equal(t, tc.want, checker(req))
		})
	}
}

// With an explicit allowlist the CORS middleware must echo the requesting
// origin rather than "*", and must advertise Vary: Origin so a shared cache
// cannot hand one site's response to another.
//
//nolint:paralleltest // setupHTTPHandler mutates global gin mode and reads GIN_MODE, which TestRun writes
func TestSetupHTTPHandlerEchoesOriginAndVaries(t *testing.T) {
	cfg := &config.Config{}
	cfg.AllowedOrigins = []string{"https://allowed.example"}
	cfg.AllowedHeaders = []string{"Content-Type", "Authorization"}
	cfg.AllowCredentials = true
	cfg.Disabled = true

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	handler := setupHTTPHandler(cfg, logger.New("error"), &usecase.Usecases{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	req.Header.Set("Origin", "https://allowed.example")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "https://allowed.example", w.Header().Get("Access-Control-Allow-Origin"))
	require.NotEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, w.Header().Values("Vary"), "Origin")
	require.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

// Browsers send an Origin header on same-origin state-changing requests too, so
// an allowlist that names only localhost must not 403 a Console reached at its
// own LAN address. gin-contrib/cors exempts same-origin before validating, and
// newOriginChecker mirrors that for the relay.
//
//nolint:paralleltest // setupHTTPHandler mutates global gin mode and reads GIN_MODE, which TestRun writes
func TestSetupHTTPHandlerAllowsSameOriginOutsideAllowlist(t *testing.T) {
	cfg := &config.Config{}
	cfg.AllowedOrigins = []string{"https://localhost:8181"}
	cfg.AllowedHeaders = []string{"Content-Type"}
	cfg.AllowCredentials = true
	cfg.Disabled = true

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	handler := setupHTTPHandler(cfg, logger.New("error"), &usecase.Usecases{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	req.Host = "10.20.224.56:8181"
	req.Header.Set("Origin", "https://10.20.224.56:8181")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// And the relay accepts the matching websocket handshake.
	require.True(t, newOriginChecker(cfg.AllowedOrigins)(req))
}

// A wildcard allowlist forbids credentials: the browser refuses the pairing, so
// leaving Access-Control-Allow-Credentials on would only break cookie auth
// while advertising it.
//
//nolint:paralleltest // setupHTTPHandler mutates global gin mode and reads GIN_MODE, which TestRun writes
func TestSetupHTTPHandlerDropsCredentialsUnderWildcard(t *testing.T) {
	cfg := &config.Config{}
	cfg.AllowedOrigins = []string{"*"}
	cfg.AllowedHeaders = []string{"Content-Type"}
	cfg.AllowCredentials = true
	cfg.Disabled = true

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	handler := setupHTTPHandler(cfg, logger.New("error"), &usecase.Usecases{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	req.Header.Set("Origin", "https://evil.example")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	require.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestRun(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	t.Cleanup(ctrl.Finish)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockHTTPServer := mocks.NewMockHTTPServer(ctrl)

	cfg, _ := config.NewConfig()
	cfg.Provider = ProviderPostgres
	cfg.DB.URL = "postgres://testuser:testpass@localhost/testdb"

	tests := []struct {
		name       string
		setupMocks func()
		setupEnv   func()
		cfg        *config.Config
		expectFunc func(t *testing.T)
	}{
		{
			name: "Successful run and shutdown",
			setupMocks: func() {
				mockDB.EXPECT().Close().Return(nil).Times(1)
				mockHTTPServer.EXPECT().Notify().Return(make(chan error)).Times(1)
				mockHTTPServer.EXPECT().Shutdown().Return(nil).Times(1)
			},
			setupEnv: func() {
				os.Setenv("GIN_MODE", "release")
			},
			cfg: cfg,
			expectFunc: func(_ *testing.T) {
				go func() {
					Run(cfg, logger.New("info"))
				}()
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.setupEnv()
			tc.setupMocks()

			tc.expectFunc(t)
		})
	}
}
