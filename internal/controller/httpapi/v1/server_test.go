package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func serverTestEngine(t *testing.T, disableCIRA bool) *gin.Engine {
	t.Helper()

	cfg := &config.Config{App: config.App{DisableCIRA: disableCIRA}}

	engine := gin.New()
	group := engine.Group("/api/v1")

	NewServerRoutes(group, cfg)

	return engine
}

func TestServerFeatures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		disableCIRA     bool
		wantCIRAEnabled bool
	}{
		{
			name:            "CIRA enabled when not disabled",
			disableCIRA:     false,
			wantCIRAEnabled: true,
		},
		{
			name:            "CIRA disabled",
			disableCIRA:     true,
			wantCIRAEnabled: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine := serverTestEngine(t, tt.disableCIRA)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/server/features", http.NoBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var features dto.ServerFeatures
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &features))
			require.Equal(t, tt.wantCIRAEnabled, features.CIRAEnabled)
		})
	}
}
