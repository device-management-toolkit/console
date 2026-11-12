package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

func TestAdoptRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		body         string
		adoptErr     error
		callsAdopt   bool
		expectedCode int
	}{
		{"adopts with valid credentials", `{"username":"admin","password":"pw"}`, nil, true, http.StatusOK},
		{"rejects a missing password", `{"username":"admin"}`, nil, false, http.StatusBadRequest},
		{"maps rejected credentials to 403", `{"username":"admin","password":"bad"}`, devices.ErrInvalidCredentials, true, http.StatusForbidden},
		{"maps an unknown device to 404", `{"username":"admin","password":"pw"}`, devices.ErrNotFound, true, http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			device, engine := devicesTest(t)

			if tc.callsAdopt {
				device.EXPECT().
					Adopt(context.Background(), "guid", "", "admin", gomock.Any()).
					Return(&dto.Device{GUID: "guid"}, tc.adoptErr)
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/devices/guid/adopt", strings.NewReader(tc.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)
		})
	}
}
