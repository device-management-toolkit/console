package devices_test

import (
	"context"
	"sync"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestRedirect(t *testing.T) {
	t.Parallel()

	mockConn := &websocket.Conn{}
	guid := "device-guid-123"
	mode := "default"

	tests := []struct {
		name        string
		setup       func(*mocks.MockRedirection, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *sync.WaitGroup)
		expectedErr error
	}{
		{
			name: "GetByID fail redirection",
			setup: func(_ *mocks.MockRedirection, mockRepo *mocks.MockDeviceManagementRepository, mockWSMAN *mocks.MockWSMAN, wg *sync.WaitGroup) {
				mockWSMAN.EXPECT().Worker().Do(func() {
					defer wg.Done()
				}).Times(1)
				mockRepo.EXPECT().GetByID(gomock.Any(), guid, "").Return(nil, ErrGeneral)
			},
			expectedErr: ErrGeneral,
		},
		{
			name: "RedirectConnect fail redirection",
			setup: func(mockRedir *mocks.MockRedirection, mockRepo *mocks.MockDeviceManagementRepository, mockWSMAN *mocks.MockWSMAN, wg *sync.WaitGroup) {
				mockWSMAN.EXPECT().Worker().Do(func() {
					defer wg.Done()
				}).Times(1)
				mockRepo.EXPECT().GetByID(gomock.Any(), guid, "").Return(&entity.Device{
					GUID:     guid,
					Username: "user",
					Password: "pass",
				}, nil)
				mockRedir.EXPECT().SetupWsmanClient(gomock.Any(), true, true).Return(wsman.Messages{})
				mockRedir.EXPECT().RedirectConnect(gomock.Any(), gomock.Any()).Return(ErrGeneral)
			},
			expectedErr: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRedirection := mocks.NewMockRedirection(ctrl)
			mockRepo := mocks.NewMockDeviceManagementRepository(ctrl)
			mockWSMAN := mocks.NewMockWSMAN(ctrl)

			var wg sync.WaitGroup

			wg.Add(1)

			tc.setup(mockRedirection, mockRepo, mockWSMAN, &wg)

			uc := devices.New(mockRepo, mockWSMAN, mockRedirection, logger.New("test"), mocks.MockCrypto{})

			wg.Wait()

			err := uc.Redirect(context.Background(), mockConn, guid, mode)

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
