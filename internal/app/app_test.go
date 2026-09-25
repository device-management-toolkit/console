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

// headerTestEngine wires the middleware onto the three response shapes the
// scanner reached: a JSON route, the SPA fallback that serves index.html, and
// an aborted request.
func headerTestEngine(permissionsPolicy string) *gin.Engine {
	r := gin.New()
	r.Use(securityHeaders(permissionsPolicy))
	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"a": 1}) })
	// SPA fallback, the path the finding was reported against.
	r.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html></html>"))
	})
	r.GET("/unauth", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	})

	return r
}

var headerTestPaths = []string{"/ok", "/no/such/route", "/unauth"}

func TestSecurityHeadersSetsNoSniff(t *testing.T) {
	t.Parallel()

	r := headerTestEngine(config.DefaultPermissionsPolicy)

	for _, path := range headerTestPaths {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))

		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), path)
	}
}

func TestSecurityHeadersSetsPermissionsPolicy(t *testing.T) {
	t.Parallel()

	r := headerTestEngine(config.DefaultPermissionsPolicy)

	for _, path := range headerTestPaths {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))

		require.Equal(t, config.DefaultPermissionsPolicy, w.Header().Get("Permissions-Policy"), path)
	}
}

// The bundled UI uses both of these; denying them outright would break the KVM
// viewer and the copy-to-clipboard buttons.
func TestDefaultPermissionsPolicyKeepsFeaturesTheUIUses(t *testing.T) {
	t.Parallel()

	for _, feature := range []string{"clipboard-write=(self)", "fullscreen=(self)"} {
		require.Contains(t, config.DefaultPermissionsPolicy, feature)
	}

	for _, denied := range []string{"camera=()", "microphone=()", "geolocation=()", "payment=()", "usb=()"} {
		require.Contains(t, config.DefaultPermissionsPolicy, denied)
	}
}

// An empty policy hands the header back to whoever serves the UI or fronts
// Console, without dropping nosniff.
func TestSecurityHeadersOmitsEmptyPermissionsPolicy(t *testing.T) {
	t.Parallel()

	r := headerTestEngine("")

	for _, path := range headerTestPaths {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))

		require.Empty(t, w.Header().Get("Permissions-Policy"), path)
		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), path)
	}
}

// The security-header middleware has to sit ahead of CORS in the engine chain: the CORS
// middleware answers preflights and rejects disallowed origins itself, so
// anything registered after it never runs on those responses.
//
//nolint:paralleltest // setupHTTPHandler mutates global gin mode and reads GIN_MODE, which TestRun writes
func TestSetupHTTPHandlerSetsNoSniffAheadOfCORS(t *testing.T) {
	cfg := &config.Config{}
	cfg.AllowedOrigins = []string{"https://allowed.example"}
	cfg.AllowedHeaders = []string{"Content-Type"}
	cfg.PermissionsPolicy = config.DefaultPermissionsPolicy
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
		require.Equal(t, config.DefaultPermissionsPolicy, w.Header().Get("Permissions-Policy"), tc.name)
	}
}

func TestRun(t *testing.T) { //nolint:tparallel // t.Setenv is incompatible with t.Parallel, so this test can't mark itself parallel even though its subtest does
	t.Setenv("AUTH_JWT_KEY", "test-jwt-key")

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
