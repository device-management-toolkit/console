package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// TestMain seeds the config singleton NewLoginRoute reads at construction time.
func TestMain(m *testing.M) {
	config.ConsoleConfig = &config.Config{Auth: config.Auth{Disabled: true, JWTKey: "test"}}

	os.Exit(m.Run())
}

// metricsStatus builds a router with the given metrics setting and reports the
// status GET /metrics answers with.
func metricsStatus(t *testing.T, disableMetrics bool) int {
	t.Helper()

	gin.SetMode(gin.TestMode)

	handler := gin.New()
	cfg := &config.Config{
		App:  config.App{DisableMetrics: disableMetrics},
		Auth: config.Auth{Disabled: true, JWTKey: "test"},
	}

	NewRouter(handler, logger.New("error"), usecase.Usecases{}, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	handler.ServeHTTP(w, req)

	return w.Code
}

// Both cases run in one test: NewRouter registers custom validators on gin's
// shared binding engine, so two routers must not be built concurrently.
func TestMetricsEndpoint(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusOK, metricsStatus(t, false), "metrics served by default")
	require.Equal(t, http.StatusNotFound, metricsStatus(t, true), "metrics removed when disabled")
}
