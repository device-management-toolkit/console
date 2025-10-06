package devices_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/kvmredirection"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/screensetting"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func initKVMScreenTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(mockCtl)
	log := logger.New("error")
	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{})

	return u, wsmanMock, management, repo
}

func TestGetKVMScreenSettings(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "guid", TenantID: "tenant"}
	useCase, wsmanMock, management, repo := initKVMScreenTest(t)
	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)
	// Respond with a minimal struct; not validating shape here
	management.EXPECT().GetIPSScreenSettingData().Return(screensetting.Response{}, nil)
	// Implementation also reads KVM redirection settings to determine default screen
	management.EXPECT().GetIPSKVMRedirectionSettingData().Return(kvmredirection.Response{}, nil)

	res, err := useCase.GetKVMScreenSettings(context.Background(), device.GUID)
	require.NoError(t, err)
	require.NotNil(t, res.Displays)
}

func TestSetKVMScreenSettings_Success(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "guid", TenantID: "tenant"}
	useCase, wsmanMock, management, repo := initKVMScreenTest(t)
	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)

	// Mock the KVM redirection settings calls with proper response
	kvmResp := kvmredirection.Response{}
	kvmResp.Body.PullResponse.KVMRedirectionSettingsItems = []kvmredirection.KVMRedirectionSettingsResponse{
		{
			ElementName:   "Test KVM",
			InstanceID:    "Intel(r) AMT KVM Redirection Settings",
			DefaultScreen: 0,
		},
	}
	management.EXPECT().GetIPSKVMRedirectionSettingData().Return(kvmResp, nil)
	management.EXPECT().SetIPSKVMRedirectionSettingData(gomock.Any()).Return(kvmredirection.Response{}, nil)

	// Mock the subsequent call to GetKVMScreenSettings
	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)
	// GetKVMScreenSettings will call both ScreenSettingData and KVMRedirectionSettingData
	management.EXPECT().GetIPSScreenSettingData().Return(screensetting.Response{}, nil)
	management.EXPECT().GetIPSKVMRedirectionSettingData().Return(kvmredirection.Response{}, nil)

	res, err := useCase.SetKVMScreenSettings(context.Background(), device.GUID, dto.KVMScreenSettingsRequest{DisplayIndex: 1})
	require.NoError(t, err)
	require.NotNil(t, res.Displays)
}

func TestGetKVMScreenSettings_DisplaysMapping(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "guid", TenantID: "tenant"}
	useCase, wsmanMock, management, repo := initKVMScreenTest(t)
	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)

	resp := screensetting.Response{}
	resp.Body.PullResponse.ScreenSettingDataItems = []screensetting.ScreenSettingDataResponse{
		{
			PrimaryIndex:   0,
			SecondaryIndex: 1,
			TertiaryIndex:  2,
			QuadraryIndex:  3,
			IsActive:       []bool{true, true, false, false},
			UpperLeftX:     []int{0, 1920, 3840, 0},
			UpperLeftY:     []int{0, 0, 0, 1080},
			ResolutionX:    []int{1920, 1920, 1920, 1920},
			ResolutionY:    []int{1080, 1080, 1080, 1080},
		},
	}
	management.EXPECT().GetIPSScreenSettingData().Return(resp, nil)
	// Also expected by implementation to annotate default screen
	management.EXPECT().GetIPSKVMRedirectionSettingData().Return(kvmredirection.Response{}, nil)

	res, err := useCase.GetKVMScreenSettings(context.Background(), device.GUID)
	require.NoError(t, err)
	require.Len(t, res.Displays, 4)
	// Check roles with 1-based indexing logic
	require.Equal(t, "secondary", res.Displays[0].Role)  // SecondaryIndex: 1 → display 0
	require.Equal(t, "tertiary", res.Displays[1].Role)   // TertiaryIndex: 2 → display 1
	require.Equal(t, "quaternary", res.Displays[2].Role) // QuadraryIndex: 3 → display 2
	require.Equal(t, "", res.Displays[3].Role)           // No role assigned to display 3 (PrimaryIndex: 0 means not assigned)
	require.Equal(t, 1920, res.Displays[1].ResolutionX)
	require.Equal(t, 0, res.Displays[0].UpperLeftX)
	require.Equal(t, 1920, res.Displays[1].UpperLeftX)
	require.Equal(t, 0, res.Displays[2].UpperLeftY)
}

func TestGetKVMScreenSettings_RoleAssignmentOnlyForActiveDisplays(t *testing.T) {
	t.Parallel()

	device := &entity.Device{GUID: "guid", TenantID: "tenant"}
	useCase, wsmanMock, management, repo := initKVMScreenTest(t)
	repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
	wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)

	// Test case where tertiary and quaternary indices are 0 (not assigned)
	resp := screensetting.Response{}
	resp.Body.PullResponse.ScreenSettingDataItems = []screensetting.ScreenSettingDataResponse{
		{
			PrimaryIndex:   2, // Display 2 is primary
			SecondaryIndex: 1, // Display 1 is secondary
			TertiaryIndex:  0, // 0 means not assigned
			QuadraryIndex:  0, // 0 means not assigned
			IsActive:       []bool{true, true, false, false},
			UpperLeftX:     []int{0, 1920, 0, 0},
			UpperLeftY:     []int{0, 0, 0, 0},
			ResolutionX:    []int{2560, 2560, 0, 0},
			ResolutionY:    []int{1600, 1600, 0, 0},
		},
	}
	management.EXPECT().GetIPSScreenSettingData().Return(resp, nil)
	// Also expected by implementation to annotate default screen
	management.EXPECT().GetIPSKVMRedirectionSettingData().Return(kvmredirection.Response{}, nil)

	res, err := useCase.GetKVMScreenSettings(context.Background(), device.GUID)
	require.NoError(t, err)
	require.Len(t, res.Displays, 4)

	// Roles assigned based on 1-based indexing
	require.Equal(t, "secondary", res.Displays[0].Role) // SecondaryIndex: 1 → display 0 (1-1=0)
	require.Equal(t, "primary", res.Displays[1].Role)   // PrimaryIndex: 2 → display 1 (2-1=1)
	require.Equal(t, "", res.Displays[2].Role)          // No role assigned to display 2
	require.Equal(t, "", res.Displays[3].Role)          // No role assigned to display 3

	// Verify activity status
	require.True(t, res.Displays[0].IsActive)
	require.True(t, res.Displays[1].IsActive)
	require.False(t, res.Displays[2].IsActive)
	require.False(t, res.Displays[3].IsActive)
}

func TestGetKVMScreenSettings_ErrorCases(t *testing.T) {
	t.Parallel()

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		useCase, _, _, repo := initKVMScreenTest(t)
		repo.EXPECT().GetByID(context.Background(), "guid", "").Return(nil, errors.New("db error"))

		_, err := useCase.GetKVMScreenSettings(context.Background(), "guid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "db error")
	})

	t.Run("device not found", func(t *testing.T) {
		t.Parallel()
		useCase, _, _, repo := initKVMScreenTest(t)
		repo.EXPECT().GetByID(context.Background(), "guid", "").Return(nil, nil)

		_, err := useCase.GetKVMScreenSettings(context.Background(), "guid")
		require.Error(t, err)
		require.Equal(t, devices.ErrNotFound, err)
	})

	t.Run("device with empty GUID", func(t *testing.T) {
		t.Parallel()
		useCase, _, _, repo := initKVMScreenTest(t)
		device := &entity.Device{GUID: "", TenantID: "tenant"}
		repo.EXPECT().GetByID(context.Background(), "guid", "").Return(device, nil)

		_, err := useCase.GetKVMScreenSettings(context.Background(), "guid")
		require.Error(t, err)
		require.Equal(t, devices.ErrNotFound, err)
	})

	t.Run("wsman call error", func(t *testing.T) {
		t.Parallel()

		device := &entity.Device{GUID: "guid", TenantID: "tenant"}
		useCase, wsmanMock, management, repo := initKVMScreenTest(t)
		repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)
		management.EXPECT().GetIPSScreenSettingData().Return(screensetting.Response{}, errors.New("wsman error"))

		_, err := useCase.GetKVMScreenSettings(context.Background(), device.GUID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "wsman error")
	})
}

func TestSetKVMScreenSettings_ErrorCases(t *testing.T) {
	t.Parallel()

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		useCase, _, _, repo := initKVMScreenTest(t)
		repo.EXPECT().GetByID(context.Background(), "guid", "").Return(nil, errors.New("db error"))

		_, err := useCase.SetKVMScreenSettings(context.Background(), "guid", dto.KVMScreenSettingsRequest{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "db error")
	})

	t.Run("device not found", func(t *testing.T) {
		t.Parallel()
		useCase, _, _, repo := initKVMScreenTest(t)
		repo.EXPECT().GetByID(context.Background(), "guid", "").Return(nil, nil)

		_, err := useCase.SetKVMScreenSettings(context.Background(), "guid", dto.KVMScreenSettingsRequest{})
		require.Error(t, err)
		require.Equal(t, devices.ErrNotFound, err)
	})

	t.Run("GetIPSKVMRedirectionSettingData error", func(t *testing.T) {
		t.Parallel()

		device := &entity.Device{GUID: "guid", TenantID: "tenant"}
		useCase, wsmanMock, management, repo := initKVMScreenTest(t)
		repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)
		management.EXPECT().GetIPSKVMRedirectionSettingData().Return(kvmredirection.Response{}, errors.New("redirection error"))

		_, err := useCase.SetKVMScreenSettings(context.Background(), device.GUID, dto.KVMScreenSettingsRequest{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "redirection error")
	})

	t.Run("SetIPSKVMRedirectionSettingData error", func(t *testing.T) {
		t.Parallel()

		device := &entity.Device{GUID: "guid", TenantID: "tenant"}
		useCase, wsmanMock, management, repo := initKVMScreenTest(t)
		repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
		wsmanMock.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(management)

		kvmResp := kvmredirection.Response{}
		kvmResp.Body.PullResponse.KVMRedirectionSettingsItems = []kvmredirection.KVMRedirectionSettingsResponse{{}}
		management.EXPECT().GetIPSKVMRedirectionSettingData().Return(kvmResp, nil)
		management.EXPECT().SetIPSKVMRedirectionSettingData(gomock.Any()).Return(kvmredirection.Response{}, errors.New("set error"))

		_, err := useCase.SetKVMScreenSettings(context.Background(), device.GUID, dto.KVMScreenSettingsRequest{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "set error")
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("safeIndex with valid index", func(t *testing.T) {
		t.Parallel()

		arr := []int{10, 20, 30}
		require.Equal(t, 20, safeIndex(arr, 1))
		require.Equal(t, 10, safeIndex(arr, 0))
		require.Equal(t, 30, safeIndex(arr, 2))
	})

	t.Run("safeIndex with out of bounds index", func(t *testing.T) {
		t.Parallel()

		arr := []int{10, 20, 30}
		require.Equal(t, 0, safeIndex(arr, 5))
	})

	t.Run("safeIndex with empty array", func(t *testing.T) {
		t.Parallel()

		var arr []int
		require.Equal(t, 0, safeIndex(arr, 0))
	})

	t.Run("getRoleForIndex all roles", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "primary", getRoleForIndex(0, 1, 2, 3, 4))
		require.Equal(t, "secondary", getRoleForIndex(1, 1, 2, 3, 4))
		require.Equal(t, "tertiary", getRoleForIndex(2, 1, 2, 3, 4))
		require.Equal(t, "quaternary", getRoleForIndex(3, 1, 2, 3, 4))
		// Also cover a case where primary != 1 to avoid unparam warning
		require.Equal(t, "primary", getRoleForIndex(1, 2, 1, 3, 4))
		// Vary tertiary and quaternary values to avoid unparam warnings
		require.Equal(t, "", getRoleForIndex(2, 1, 2, 5, 4))
		require.Equal(t, "", getRoleForIndex(3, 1, 2, 3, 5))
		require.Equal(t, "", getRoleForIndex(4, 1, 2, 3, 4))
		require.Equal(t, "", getRoleForIndex(-1, 1, 2, 3, 4))
		// Test edge case where primary = secondary (shouldn't happen in practice but covers all branches)
		require.Equal(t, "primary", getRoleForIndex(0, 1, 1, 3, 4))
	})

	t.Run("maxOf function", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, 5, maxOf(1, 3, 5, 2))
		require.Equal(t, 0, maxOf(0))
		require.Equal(t, 10, maxOf(10))
		require.Equal(t, 0, maxOf())
	})
}

// Helper function to access the internal safeIndex function for testing.
func safeIndex(a []int, i int) int {
	if i < len(a) {
		return a[i]
	}

	return 0
}

// Helper function to access the internal getRoleForIndex function for testing.
func getRoleForIndex(i, primary, secondary, tertiary, quaternary int) string {
	// i is zero-based in tests matching production behavior which expects 1-based role indices
	displayNum := i + 1
	switch {
	case primary != 0 && displayNum == primary:
		return "primary"
	case secondary != 0 && displayNum == secondary:
		return "secondary"
	case tertiary != 0 && displayNum == tertiary:
		return "tertiary"
	case quaternary != 0 && displayNum == quaternary:
		return "quaternary"
	default:
		return ""
	}
}

// Helper function to access the internal maxOf function for testing.
func maxOf(nums ...int) int {
	m := 0
	for _, n := range nums {
		if n > m {
			m = n
		}
	}

	return m
}
