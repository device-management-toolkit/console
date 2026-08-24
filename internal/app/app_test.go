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

	tests := []struct {
		name   string
		allow  []string
		origin string
		want   bool
	}{
		{name: "wildcard allows any origin", allow: []string{"*"}, origin: "https://evil.example", want: true},
		{name: "explicit origin is allowed", allow: []string{"https://allowed.example"}, origin: "https://allowed.example", want: true},
		{name: "different origin is rejected", allow: []string{"https://allowed.example"}, origin: "https://evil.example", want: false},
		{name: "missing origin is allowed for non-browser clients", allow: []string{"https://allowed.example"}, origin: "", want: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			checker := newOriginChecker(tc.allow)
			req := httptest.NewRequest(http.MethodGet, "/relay/webrelay.ashx", http.NoBody)

			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			require.Equal(t, tc.want, checker(req))
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
