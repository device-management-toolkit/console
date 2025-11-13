package app_test

import (
	"os"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/app"
	"github.com/device-management-toolkit/console/internal/mocks"
)

func TestRun(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	t.Cleanup(ctrl.Finish)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockHTTPServer := mocks.NewMockHTTPServer(ctrl)

	cfg, _ := config.NewConfig()
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
					app.Run(cfg)
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
