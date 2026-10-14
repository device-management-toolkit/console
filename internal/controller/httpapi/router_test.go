package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// The UI calls server/features before login to learn the auth mode, so it must
// stay reachable without a token in both modes.
//
//nolint:paralleltest // NewLoginRoute reads the global config.ConsoleConfig
func TestServerFeaturesIsPublic(t *testing.T) {
	tests := []struct {
		name         string
		authDisabled bool
	}{
		{"auth enabled", false},
		{"auth disabled", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Disabled = tc.authDisabled
			cfg.JWTKey = "test-key"

			prev := config.ConsoleConfig

			t.Cleanup(func() { config.ConsoleConfig = prev })

			config.ConsoleConfig = cfg

			engine := gin.New()
			NewRouter(engine, logger.New("error"), usecase.Usecases{}, cfg)

			if !tc.authDisabled {
				// Sanity check that the JWT middleware is active for protected routes.
				w := httptest.NewRecorder()
				engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/devices", http.NoBody))
				require.Equal(t, http.StatusUnauthorized, w.Code)
			}

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/server/features", http.NoBody))
			require.Equal(t, http.StatusOK, w.Code)

			var features dto.ServerFeatures
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &features))
			require.Equal(t, tc.authDisabled, features.AuthDisabled)
		})
	}
}
