package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// APIVerifiesJWT() is read while building routes, so construction alone covers both branches.
//
//nolint:paralleltest // mutates the shared config.ConsoleConfig global
func TestNewRouterGatewayEnforced(t *testing.T) {
	cfg := &config.Config{}
	cfg.GatewayEnforced = true

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	engine := gin.New()
	NewRouter(engine, logger.New("error"), usecase.Usecases{}, cfg)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody))
	require.Equal(t, http.StatusOK, w.Code)
}

//nolint:paralleltest // mutates the shared config.ConsoleConfig global
func TestNewRouterDefaultChecksJWT(t *testing.T) {
	cfg := &config.Config{}

	prev := config.ConsoleConfig

	t.Cleanup(func() { config.ConsoleConfig = prev })

	config.ConsoleConfig = cfg

	NewRouter(gin.New(), logger.New("error"), usecase.Usecases{}, cfg)
}
