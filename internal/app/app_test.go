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

// The shipped default (empty list) must emit no CORS headers at all.
func TestCORSMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		origins       []string
		credentials   bool
		origin        string
		wantCode      int
		wantAllow     string
		wantVary      string
		wantCredsHdr  string
		wantNoHeaders bool
	}{
		{
			name:          "empty list emits no CORS headers",
			origins:       nil,
			origin:        "https://evil.example",
			wantCode:      http.StatusOK,
			wantNoHeaders: true,
		},
		{
			name:      "wildcard allows any origin without credentials",
			origins:   []string{"*"},
			origin:    "https://evil.example",
			wantCode:  http.StatusOK,
			wantAllow: "*",
		},
		{
			name:         "wildcard ignores allow_credentials",
			origins:      []string{"*"},
			credentials:  true,
			origin:       "https://evil.example",
			wantCode:     http.StatusOK,
			wantAllow:    "*",
			wantCredsHdr: "",
		},
		{
			name:         "explicit list echoes origin with Vary",
			origins:      []string{"https://ui.example"},
			credentials:  true,
			origin:       "https://ui.example",
			wantCode:     http.StatusOK,
			wantAllow:    "https://ui.example",
			wantVary:     "Origin",
			wantCredsHdr: "true",
		},
		{
			name:     "explicit list rejects other origins",
			origins:  []string{"https://ui.example"},
			origin:   "https://evil.example",
			wantCode: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			cfg.AllowedOrigins = tc.origins
			cfg.AllowedHeaders = []string{"Content-Type"}
			cfg.AllowCredentials = tc.credentials

			r := gin.New()

			mw := corsMiddleware(cfg, logger.New("error"))
			if len(tc.origins) == 0 {
				require.Nil(t, mw)
			} else {
				require.NotNil(t, mw)
				r.Use(mw)
			}

			r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/ok", http.NoBody)
			req.Header.Set("Origin", tc.origin)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantCode, w.Code)

			if tc.wantNoHeaders {
				for name := range w.Header() {
					require.NotContains(t, name, "Access-Control-", "unexpected CORS header %s", name)
				}

				require.Empty(t, w.Header().Get("Vary"))

				return
			}

			require.Equal(t, tc.wantAllow, w.Header().Get("Access-Control-Allow-Origin"))
			require.Equal(t, tc.wantVary, w.Header().Get("Vary"))
			require.Equal(t, tc.wantCredsHdr, w.Header().Get("Access-Control-Allow-Credentials"))
		})
	}
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
