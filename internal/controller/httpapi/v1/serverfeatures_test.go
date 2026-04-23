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

func newServerFeaturesTestEngine(t *testing.T, cfg *config.Config) *gin.Engine {
	t.Helper()

	engine := gin.New()
	sfr := NewServerFeaturesRoute(cfg)
	engine.GET("/api/v1/features", sfr.Handler)

	return engine
}

func TestServerFeaturesRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		disableCIRA bool
		wantCIRA    bool
	}{
		{
			name:        "CIRA enabled",
			disableCIRA: false,
			wantCIRA:    true,
		},
		{
			name:        "CIRA disabled",
			disableCIRA: true,
			wantCIRA:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			cfg.DisableCIRA = tc.disableCIRA

			engine := newServerFeaturesTestEngine(t, cfg)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/features", http.NoBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var got dto.ServerFeaturesResponse

			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			require.Equal(t, tc.wantCIRA, got.CIRA)
		})
	}
}
