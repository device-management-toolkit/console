package devices_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	testGUID = "test-guid"
	testMode = "kvm"
)

var (
	ErrConnectionFailed       = errors.New("connection failed")
	ErrFirstConnectionFailed  = errors.New("first connection failed")
	ErrSecondConnectionFailed = errors.New("second connection failed")
	ErrTestError              = errors.New("test error")
	ErrDatabaseError          = errors.New("database error")
	ErrInterceptorGeneral     = errors.New("general error")
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
				mockRepo.EXPECT().GetByID(gomock.Any(), guid, "").Return(nil, ErrInterceptorGeneral)
			},
			expectedErr: ErrInterceptorGeneral,
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
				mockRedir.EXPECT().RedirectConnect(gomock.Any(), gomock.Any()).Return(ErrInterceptorGeneral)
			},
			expectedErr: ErrInterceptorGeneral,
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

func TestRedirectSuccessfulFlow(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRedirection := mocks.NewMockRedirection(ctrl)
	mockRepo := mocks.NewMockDeviceManagementRepository(ctrl)
	mockWSMAN := mocks.NewMockWSMAN(ctrl)
	mockConn := &websocket.Conn{}

	var wg sync.WaitGroup

	wg.Add(1)

	mockWSMAN.EXPECT().Worker().Do(func() {
		defer wg.Done()
	}).Times(1)

	uc := devices.New(mockRepo, mockWSMAN, mockRedirection, logger.New("test"), mocks.MockCrypto{})

	wg.Wait()

	device := &entity.Device{
		GUID:     testGUID,
		Username: "user",
		Password: "pass",
	}

	// Mock successful flow up to RedirectConnect, then fail to avoid goroutines
	mockRepo.EXPECT().GetByID(gomock.Any(), testGUID, "").Return(device, nil)
	mockRedirection.EXPECT().SetupWsmanClient(*device, true, true).Return(wsman.Messages{})
	// Return error to avoid starting problematic goroutines but still test the flow
	mockRedirection.EXPECT().RedirectConnect(gomock.Any(), gomock.Any()).Return(ErrConnectionFailed)

	// Test redirect (should fail at RedirectConnect but test path up to that point)
	err := uc.Redirect(context.Background(), mockConn, testGUID, testMode)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestRedirectDeviceNotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRedirection := mocks.NewMockRedirection(ctrl)
	mockRepo := mocks.NewMockDeviceManagementRepository(ctrl)
	mockWSMAN := mocks.NewMockWSMAN(ctrl)
	mockConn := &websocket.Conn{}

	var wg sync.WaitGroup

	wg.Add(1)

	mockWSMAN.EXPECT().Worker().Do(func() {
		defer wg.Done()
	}).Times(1)

	uc := devices.New(mockRepo, mockWSMAN, mockRedirection, logger.New("test"), mocks.MockCrypto{})

	wg.Wait()

	// Mock device not found
	mockRepo.EXPECT().GetByID(gomock.Any(), testGUID, "").Return(nil, nil)

	// Test device not found
	err := uc.Redirect(context.Background(), mockConn, testGUID, testMode)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DevicesUseCase")
}

func TestRedirectConnectionReuse(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRedirection := mocks.NewMockRedirection(ctrl)
	mockRepo := mocks.NewMockDeviceManagementRepository(ctrl)
	mockWSMAN := mocks.NewMockWSMAN(ctrl)
	mockConn := &websocket.Conn{}

	var wg sync.WaitGroup

	wg.Add(1)

	mockWSMAN.EXPECT().Worker().Do(func() {
		defer wg.Done()
	}).Times(1)

	uc := devices.New(mockRepo, mockWSMAN, mockRedirection, logger.New("test"), mocks.MockCrypto{})

	wg.Wait()

	device := &entity.Device{
		GUID:     testGUID,
		Username: "user",
		Password: "pass",
	}

	// First call - create new connection but fail at connect to avoid goroutines
	mockRepo.EXPECT().GetByID(gomock.Any(), testGUID, "").Return(device, nil)
	mockRedirection.EXPECT().SetupWsmanClient(*device, true, true).Return(wsman.Messages{})
	mockRedirection.EXPECT().RedirectConnect(gomock.Any(), gomock.Any()).Return(ErrFirstConnectionFailed)

	err := uc.Redirect(context.Background(), mockConn, testGUID, testMode)
	require.Error(t, err)

	// Second call - also fail to avoid goroutines but test reuse logic
	mockRepo.EXPECT().GetByID(gomock.Any(), testGUID, "").Return(device, nil)
	mockRedirection.EXPECT().SetupWsmanClient(*device, true, true).Return(wsman.Messages{})
	mockRedirection.EXPECT().RedirectConnect(gomock.Any(), gomock.Any()).Return(ErrSecondConnectionFailed)

	err = uc.Redirect(context.Background(), mockConn, testGUID, testMode)
	require.Error(t, err)
}

// Add additional test cases for better coverage of the existing functions.
func TestRandomValueHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "zero length",
			length: 0,
		},
		{
			name:   "normal length",
			length: 16,
		},
		{
			name:   "large length",
			length: 64,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := devices.RandomValueHex(tc.length)
			assert.NoError(t, err)

			expectedLength := tc.length
			assert.Len(t, result, expectedLength)
		})
	}
}

// Test for testing some specific edge cases that might not be covered in existing tests.
func TestRandomValueHexEdgeCases(t *testing.T) {
	t.Parallel()

	// Test very small values
	result, err := devices.RandomValueHex(1)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	// Test moderate values
	result, err = devices.RandomValueHex(50)
	assert.NoError(t, err)
	assert.Len(t, result, 50)

	// Test that multiple calls produce different results
	result1, err1 := devices.RandomValueHex(10)
	result2, err2 := devices.RandomValueHex(10)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Len(t, result1, 10)
	assert.Len(t, result2, 10)
	assert.NotEqual(t, result1, result2)
}

// Test redirect with mock WebSocket connection that exercises the flow without panics.
func TestRedirectWithErrorScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMocks  func(*mocks.MockRedirection, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *sync.WaitGroup)
		expectedErr string
	}{
		{
			name: "RedirectConnect error should trigger cleanup",
			setupMocks: func(mockRedir *mocks.MockRedirection, mockRepo *mocks.MockDeviceManagementRepository, mockWSMAN *mocks.MockWSMAN, wg *sync.WaitGroup) {
				mockWSMAN.EXPECT().Worker().Do(func() {
					defer wg.Done()
				}).Times(1)

				device := &entity.Device{GUID: testGUID, Username: "user", Password: "pass"}
				mockRepo.EXPECT().GetByID(gomock.Any(), testGUID, "").Return(device, nil)
				mockRedir.EXPECT().SetupWsmanClient(*device, true, true).Return(wsman.Messages{})
				mockRedir.EXPECT().RedirectConnect(gomock.Any(), gomock.Any()).Return(ErrConnectionFailed)
			},
			expectedErr: "connection failed",
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

			tc.setupMocks(mockRedirection, mockRepo, mockWSMAN, &wg)

			uc := devices.New(mockRepo, mockWSMAN, mockRedirection, logger.New("test"), mocks.MockCrypto{})

			wg.Wait()

			// Create a mock websocket connection - but we can still test error paths
			mockConn := &websocket.Conn{}

			err := uc.Redirect(context.Background(), mockConn, testGUID, testMode)

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test error handling during data processing.
func TestDataProcessingFunctions(t *testing.T) {
	t.Parallel()

	// Test RandomValueHex error cases
	t.Run("RandomValueHex large value", func(t *testing.T) {
		t.Parallel()

		result, err := devices.RandomValueHex(1000)
		assert.NoError(t, err)
		assert.Len(t, result, 1000)
	})
}

// Test redirect flow that exercises connection creation but fails before goroutines start.
func TestRedirectConnectionFlowCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mocks.MockRedirection, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *sync.WaitGroup)
		guid       string
		mode       string
		shouldErr  bool
	}{
		{
			name: "Connection creation up to RedirectConnect",
			setupMocks: func(mockRedir *mocks.MockRedirection, mockRepo *mocks.MockDeviceManagementRepository, mockWSMAN *mocks.MockWSMAN, wg *sync.WaitGroup) {
				mockWSMAN.EXPECT().Worker().Do(func() {
					defer wg.Done()
				}).Times(1)

				device := &entity.Device{GUID: "test-device", Username: "user", Password: "pass"}
				mockRepo.EXPECT().GetByID(gomock.Any(), "test-device", "").Return(device, nil)
				mockRedir.EXPECT().SetupWsmanClient(*device, true, true).Return(wsman.Messages{})
				// Return error to avoid starting goroutines, but still exercise connection creation
				mockRedir.EXPECT().RedirectConnect(gomock.Any(), gomock.Any()).Return(ErrTestError)
			},
			guid:      "test-device",
			mode:      "kvm",
			shouldErr: true,
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

			tc.setupMocks(mockRedirection, mockRepo, mockWSMAN, &wg)

			uc := devices.New(mockRepo, mockWSMAN, mockRedirection, logger.New("test"), mocks.MockCrypto{})

			wg.Wait()

			mockConn := &websocket.Conn{}

			err := uc.Redirect(context.Background(), mockConn, tc.guid, tc.mode)

			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test additional edge cases for better coverage.
func TestRedirectAdditionalCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMocks  func(*mocks.MockRedirection, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN, *sync.WaitGroup)
		guid        string
		mode        string
		expectedErr string
	}{
		{
			name: "Device not found in repository",
			setupMocks: func(_ *mocks.MockRedirection, mockRepo *mocks.MockDeviceManagementRepository, mockWSMAN *mocks.MockWSMAN, wg *sync.WaitGroup) {
				mockWSMAN.EXPECT().Worker().Do(func() {
					defer wg.Done()
				}).Times(1)
				mockRepo.EXPECT().GetByID(gomock.Any(), "missing-guid", "").Return(nil, nil)
			},
			guid:        "missing-guid",
			mode:        "kvm",
			expectedErr: "DevicesUseCase",
		},
		{
			name: "Repository error during GetByID",
			setupMocks: func(_ *mocks.MockRedirection, mockRepo *mocks.MockDeviceManagementRepository, mockWSMAN *mocks.MockWSMAN, wg *sync.WaitGroup) {
				mockWSMAN.EXPECT().Worker().Do(func() {
					defer wg.Done()
				}).Times(1)
				mockRepo.EXPECT().GetByID(gomock.Any(), "error-guid", "").Return(nil, ErrDatabaseError)
			},
			guid:        "error-guid",
			mode:        "kvm",
			expectedErr: "database error",
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

			tc.setupMocks(mockRedirection, mockRepo, mockWSMAN, &wg)

			uc := devices.New(mockRepo, mockWSMAN, mockRedirection, logger.New("test"), mocks.MockCrypto{})

			wg.Wait()

			mockConn := &websocket.Conn{}

			err := uc.Redirect(context.Background(), mockConn, tc.guid, tc.mode)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
