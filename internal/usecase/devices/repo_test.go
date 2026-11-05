package devices_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/repoerrors"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func ptr(s string) *string {
	return &s
}

func boolPtr(v bool) *bool {
	return &v
}

// insertedDevice matches the entity passed to repo.Insert, asserting the
// server-set ID/CreatedDate/LastUpdated are populated (LastUpdated == CreatedDate)
// while ignoring their non-deterministic values.
type insertedDevice struct{ want *entity.Device }

func (m insertedDevice) Matches(x any) bool {
	got, ok := x.(*entity.Device)
	if !ok || got == nil {
		return false
	}

	if got.ID == "" || got.CreatedDate == "" {
		return false
	}

	if got.LastUpdated != got.CreatedDate {
		return false
	}

	cp := *got
	cp.ID = ""
	cp.CreatedDate = ""
	cp.LastUpdated = ""

	return reflect.DeepEqual(&cp, m.want)
}

func (m insertedDevice) String() string {
	return fmt.Sprintf("matches %+v (ignoring server-set ID/CreatedDate/LastUpdated)", m.want)
}

// updatedDevice matches the entity passed to repo.Update, asserting the
// server-set LastUpdated was refreshed (non-empty) while ignoring its value.
type updatedDevice struct{ want *entity.Device }

func (m updatedDevice) Matches(x any) bool {
	got, ok := x.(*entity.Device)
	if !ok || got == nil {
		return false
	}

	if got.LastUpdated == "" {
		return false
	}

	cp := *got
	cp.LastUpdated = ""

	return reflect.DeepEqual(&cp, m.want)
}

func (m updatedDevice) String() string {
	return fmt.Sprintf("matches %+v (ignoring server-set LastUpdated)", m.want)
}

type testUsecase struct {
	name     string
	guid     string
	tenantID string
	top      int
	skip     int
	mock     func(*mocks.MockDeviceManagementRepository, *mocks.MockWSMAN)
	res      interface{}
	err      error
}

func devicesTest(t *testing.T) (*devices.UseCase, *mocks.MockDeviceManagementRepository, *mocks.MockWSMAN) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	log := logger.New("error")
	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{})

	return u, repo, wsmanMock
}

func TestGetCount(t *testing.T) {
	t.Parallel()

	tests := []testUsecase{
		{
			name: "empty result",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, nil)
			},
			res: 0,
			err: nil,
		},
		{
			name: "result with error",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().GetCount(context.Background(), "").Return(0, devices.ErrDatabase)
			},
			res: 0,
			err: devices.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			res, err := useCase.GetCount(context.Background(), tc.tenantID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	testDevices := []entity.Device{
		{
			GUID:     "guid-123",
			TenantID: "tenant-id-456",
		},
		{
			GUID:     "guid-456",
			TenantID: "tenant-id-456",
		},
	}

	testDeviceDTOs := []dto.Device{
		{
			GUID:     "guid-123",
			TenantID: "tenant-id-456",
			Tags:     nil,
		},
		{
			GUID:     "guid-456",
			TenantID: "tenant-id-456",
			Tags:     nil,
		},
	}

	tests := []testUsecase{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					Get(context.Background(), 10, 0, "tenant-id-456").
					Return(testDevices, nil)
			},
			res: testDeviceDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					Get(context.Background(), 5, 0, "tenant-id-456").
					Return(nil, devices.ErrDatabase)
			},
			res: []dto.Device(nil),
			err: devices.ErrDatabase,
		},
		{
			name:     "zero results",
			top:      10,
			skip:     20,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					Get(context.Background(), 10, 20, "tenant-id-456").
					Return([]entity.Device{}, nil)
			},
			res: []dto.Device{},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			results, err := useCase.Get(context.Background(), tc.top, tc.skip, tc.tenantID)

			require.Equal(t, tc.res, results)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetActivated(t *testing.T) {
	t.Parallel()

	testDevices := []entity.Device{{GUID: "guid-123", TenantID: "tenant-id-456"}}
	testDeviceDTOs := []dto.Device{{GUID: "guid-123", TenantID: "tenant-id-456", Tags: nil}}

	tests := []testUsecase{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().GetActivated(context.Background(), 10, 0, "tenant-id-456").Return(testDevices, nil)
			},
			res: testDeviceDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().GetActivated(context.Background(), 5, 0, "tenant-id-456").Return(nil, devices.ErrDatabase)
			},
			res: []dto.Device(nil),
			err: devices.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			results, err := useCase.GetActivated(context.Background(), tc.top, tc.skip, tc.tenantID)

			require.Equal(t, tc.res, results)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetDiscovered(t *testing.T) {
	t.Parallel()

	testDevices := []entity.Device{{GUID: "guid-123", TenantID: "tenant-id-456"}}
	testDeviceDTOs := []dto.Device{{GUID: "guid-123", TenantID: "tenant-id-456", Tags: nil}}

	tests := []testUsecase{
		{
			name:     "successful retrieval",
			top:      10,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().GetDiscovered(context.Background(), 10, 0, "tenant-id-456").Return(testDevices, nil)
			},
			res: testDeviceDTOs,
			err: nil,
		},
		{
			name:     "database error",
			top:      5,
			skip:     0,
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().GetDiscovered(context.Background(), 5, 0, "tenant-id-456").Return(nil, devices.ErrDatabase)
			},
			res: []dto.Device(nil),
			err: devices.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			results, err := useCase.GetDiscovered(context.Background(), tc.top, tc.skip, tc.tenantID)

			require.Equal(t, tc.res, results)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetDeviceStateCounts(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval", func(t *testing.T) {
		t.Parallel()

		useCase, repo, management := devicesTest(t)
		_ = management

		repo.EXPECT().GetDeviceStateCounts(context.Background(), "tenant-id-456").Return(3, 1, nil)

		activated, discovered, err := useCase.GetDeviceStateCounts(context.Background(), "tenant-id-456")
		require.NoError(t, err)
		require.Equal(t, 3, activated)
		require.Equal(t, 1, discovered)
	})

	t.Run("database error", func(t *testing.T) {
		t.Parallel()

		useCase, repo, management := devicesTest(t)
		_ = management

		repo.EXPECT().GetDeviceStateCounts(context.Background(), "tenant-id-456").Return(0, 0, devices.ErrDatabase)

		_, _, err := useCase.GetDeviceStateCounts(context.Background(), "tenant-id-456")
		require.Error(t, err)
		require.Contains(t, err.Error(), devices.ErrDatabase.Error())
	})
}

func TestGetByID(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}
	deviceDTO := &dto.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
		Tags:     nil,
	}

	tests := []testUsecase{
		{
			name:     "successful retrieval",
			guid:     "device-guid-123",
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					GetByID(context.Background(), "device-guid-123", "tenant-id-456").
					Return(device, nil)
			},
			res: deviceDTO,
			err: nil,
		},
		{
			name:     "device not found",
			guid:     "device-guid-unknown",
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					GetByID(context.Background(), "device-guid-unknown", "tenant-id-456").
					Return(nil, nil)
			},
			res: nil,
			err: devices.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			got, err := useCase.GetByID(context.Background(), tc.guid, tc.tenantID, false)

			if tc.err != nil {
				require.Error(t, err)

				var notFoundErr repoerrors.NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.res, got)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []testUsecase{
		{
			name:     "successful deletion",
			guid:     "guid-123",
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					SoftDelete(context.Background(), "guid-123", "tenant-id-456").
					Return(true, nil)
			},
			err: nil,
		},
		{
			name:     "deletion fails - device not found",
			guid:     "guid-456",
			tenantID: "tenant-id-456",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					SoftDelete(context.Background(), "guid-456", "tenant-id-456").
					Return(false, nil)
			},
			err: devices.ErrNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			err := useCase.Delete(context.Background(), tc.guid, tc.tenantID)

			if tc.err != nil {
				require.Error(t, err)

				var notFoundErr repoerrors.NotFoundError
				require.ErrorAs(t, err, &notFoundErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		ConnectionType: entity.ConnectionTypeDirect,
		GUID:           "device-guid-123",
		TenantID:       "tenant-id-456",
		Password:       "encrypted",
		MPSPassword:    nil,
		MEBXPassword:   nil,
		Tags:           "hello,test",
	}

	deviceDTO := &dto.Device{
		ConnectionType: entity.ConnectionTypeDirect,
		GUID:           "device-guid-123",
		TenantID:       "tenant-id-456",
		Tags:           []string{"hello", "test"},
	}

	tests := []testUsecase{
		{
			name: "successful update",
			mock: func(repo *mocks.MockDeviceManagementRepository, management *mocks.MockWSMAN) {
				repo.EXPECT().
					Update(context.Background(), updatedDevice{device}).
					Return(true, nil)
				repo.EXPECT().
					GetByID(context.Background(), "device-guid-123", "tenant-id-456").
					Return(device, nil)
				management.EXPECT().
					DestroyWsmanClient(*deviceDTO)
			},
			res: deviceDTO,
			err: nil,
		},
		{
			name: "update fails - not found",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					Update(context.Background(), updatedDevice{device}).
					Return(false, nil)
			},
			res: (*dto.Device)(nil),
			err: devices.ErrNotFound,
		},
		{
			name: "update fails - database error",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					Update(context.Background(), updatedDevice{device}).
					Return(false, devices.ErrDatabase)
			},
			res: (*dto.Device)(nil),
			err: devices.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			result, err := useCase.Update(context.Background(), deviceDTO, nil)

			require.Equal(t, tc.res, result)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestInsert(t *testing.T) {
	t.Parallel()

	tests := []testUsecase{
		{
			name: "successful insertion",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				device := &entity.Device{
					ConnectionType: entity.ConnectionTypeDirect,
					GUID:           "device-guid-123",
					Password:       "encrypted",
					MPSPassword:    nil,
					MEBXPassword:   nil,
					TenantID:       "tenant-id-456",
				}

				repo.EXPECT().
					Insert(context.Background(), insertedDevice{want: device}).
					Return("unique-device-id", nil)
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "tenant-id-456").
					Return(device, nil)
			},
			res: nil, // little bit different in that the expectation is handled in the loop
			err: nil,
		},
		{
			name: "insertion fails - database error",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				device := &entity.Device{
					ConnectionType: entity.ConnectionTypeDirect,
					GUID:           "device-guid-123",
					Password:       "encrypted",
					MPSPassword:    nil,
					MEBXPassword:   nil,
					TenantID:       "tenant-id-456",
				}

				repo.EXPECT().
					Insert(context.Background(), insertedDevice{want: device}).
					Return("", devices.ErrDatabase)
			},
			res: (*dto.Device)(nil),
			err: devices.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			tc.mock(repo, management)

			deviceDTO := &dto.Device{
				GUID:     "device-guid-123",
				TenantID: "tenant-id-456",
				Tags:     []string{""},
			}

			insertedDevice, err := useCase.Insert(context.Background(), deviceDTO)

			if tc.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err.Error())
				require.Equal(t, tc.res, insertedDevice)
			} else {
				require.NoError(t, err)
				require.Equal(t, deviceDTO.TenantID, insertedDevice.TenantID)
				require.NotEmpty(t, deviceDTO.GUID)
			}
		})
	}
}

func TestUpdateWithPasswords(t *testing.T) {
	t.Parallel()

	// Entity with encrypted passwords (what gets stored in DB)
	deviceWithPasswords := &entity.Device{
		ConnectionType: entity.ConnectionTypeDirect,
		GUID:           "device-guid-123",
		TenantID:       "tenant-id-456",
		Password:       "encrypted",
		MPSPassword:    ptr("encrypted"),
		MEBXPassword:   ptr("encrypted"),
		Tags:           "hello,test",
	}

	// DTO with plaintext passwords (what comes from API)
	deviceDTOWithPasswords := &dto.Device{
		GUID:         "device-guid-123",
		TenantID:     "tenant-id-456",
		Tags:         []string{"hello", "test"},
		MPSPassword:  "mpspass",
		MEBXPassword: "mebxpass",
	}

	// Expected DTO result (passwords not returned without includeSecrets)
	expectedDTO := &dto.Device{
		ConnectionType: entity.ConnectionTypeDirect,
		GUID:           "device-guid-123",
		TenantID:       "tenant-id-456",
		Tags:           []string{"hello", "test"},
		MPSPassword:    "encrypted",
		MEBXPassword:   "encrypted",
	}

	t.Run("successful update with passwords", func(t *testing.T) {
		t.Parallel()

		useCase, repo, management := devicesTest(t)

		repo.EXPECT().
			Update(context.Background(), updatedDevice{deviceWithPasswords}).
			Return(true, nil)
		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(deviceWithPasswords, nil)
		management.EXPECT().
			DestroyWsmanClient(*expectedDTO)

		result, err := useCase.Update(context.Background(), deviceDTOWithPasswords, nil)

		require.NoError(t, err)
		require.Equal(t, expectedDTO, result)
	})
}

func TestInsertWithPasswords(t *testing.T) {
	t.Parallel()

	t.Run("successful insertion with passwords", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		// Entity with encrypted passwords
		deviceWithPasswords := &entity.Device{
			ConnectionType: entity.ConnectionTypeDirect,
			GUID:           "device-guid-123",
			TenantID:       "tenant-id-456",
			Password:       "encrypted",
			MPSPassword:    ptr("encrypted"),
			MEBXPassword:   ptr("encrypted"),
		}

		repo.EXPECT().
			Insert(context.Background(), insertedDevice{want: deviceWithPasswords}).
			Return("unique-device-id", nil)
		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(deviceWithPasswords, nil)

		// DTO with plaintext passwords
		deviceDTO := &dto.Device{
			GUID:         "device-guid-123",
			TenantID:     "tenant-id-456",
			Tags:         []string{""},
			MPSPassword:  "mpspass",
			MEBXPassword: "mebxpass",
		}

		insertedDevice, err := useCase.Insert(context.Background(), deviceDTO)

		require.NoError(t, err)
		require.Equal(t, deviceDTO.TenantID, insertedDevice.TenantID)
		require.Equal(t, "encrypted", insertedDevice.MPSPassword)
		require.Equal(t, "encrypted", insertedDevice.MEBXPassword)
	})
}

func TestGetByIDWithSecrets(t *testing.T) {
	t.Parallel()

	// Entity with encrypted passwords from DB
	deviceWithPasswords := &entity.Device{
		GUID:         "device-guid-123",
		TenantID:     "tenant-id-456",
		Password:     "encrypted",
		MPSPassword:  ptr("encrypted"),
		MEBXPassword: ptr("encrypted"),
	}

	// Expected DTO with decrypted passwords
	expectedDTO := &dto.Device{
		GUID:         "device-guid-123",
		TenantID:     "tenant-id-456",
		Tags:         nil,
		Password:     "decrypted",
		MPSPassword:  "decrypted",
		MEBXPassword: "decrypted",
	}

	t.Run("successful retrieval with secrets", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(deviceWithPasswords, nil)

		got, err := useCase.GetByID(context.Background(), "device-guid-123", "tenant-id-456", true)

		require.NoError(t, err)
		require.Equal(t, expectedDTO, got)
	})

	t.Run("retrieval with secrets - nil passwords", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		// Entity with nil passwords
		deviceNilPasswords := &entity.Device{
			GUID:         "device-guid-123",
			TenantID:     "tenant-id-456",
			Password:     "encrypted",
			MPSPassword:  nil,
			MEBXPassword: nil,
		}

		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(deviceNilPasswords, nil)

		// Expected DTO - passwords should be empty strings when nil
		expectedDTONilPasswords := &dto.Device{
			GUID:         "device-guid-123",
			TenantID:     "tenant-id-456",
			Tags:         nil,
			Password:     "decrypted",
			MPSPassword:  "",
			MEBXPassword: "",
		}

		got, err := useCase.GetByID(context.Background(), "device-guid-123", "tenant-id-456", true)

		require.NoError(t, err)
		require.Equal(t, expectedDTONilPasswords, got)
	})
}

func TestGetByID_UUIDNormalization(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "aaf0c395-c2a2-992e-5655-48210b50d8c9",
		TenantID: "tenant-id-456",
	}

	tests := []struct {
		name       string
		inputGUID  string
		expectGUID string
	}{
		{
			name:       "uppercase UUID is normalized to lowercase",
			inputGUID:  "AAF0C395-C2A2-992E-5655-48210B50D8C9",
			expectGUID: "aaf0c395-c2a2-992e-5655-48210b50d8c9",
		},
		{
			name:       "lowercase UUID stays lowercase",
			inputGUID:  "aaf0c395-c2a2-992e-5655-48210b50d8c9",
			expectGUID: "aaf0c395-c2a2-992e-5655-48210b50d8c9",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, _ := devicesTest(t)

			// Expect the normalized (lowercase) GUID to be passed to the repository
			repo.EXPECT().
				GetByID(context.Background(), tc.expectGUID, "tenant-id-456").
				Return(device, nil)

			got, err := useCase.GetByID(context.Background(), tc.inputGUID, "tenant-id-456", false)

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tc.expectGUID, got.GUID)
		})
	}
}

// TestDelete_UUIDNormalization tests that UUID is normalized for delete operations.
func TestDelete_UUIDNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		inputGUID  string
		expectGUID string
	}{
		{
			name:       "uppercase UUID is normalized to lowercase",
			inputGUID:  "AAF0C395-C2A2-992E-5655-48210B50D8C9",
			expectGUID: "aaf0c395-c2a2-992e-5655-48210b50d8c9",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, _ := devicesTest(t)

			// Expect the normalized (lowercase) GUID to be passed to the repository
			repo.EXPECT().
				SoftDelete(context.Background(), tc.expectGUID, "tenant-id-456").
				Return(true, nil)

			err := useCase.Delete(context.Background(), tc.inputGUID, "tenant-id-456")

			require.NoError(t, err)
		})
	}
}

// TestUpdate_UUIDNormalization tests that UUID is normalized for update operations.
func TestUpdate_UUIDNormalization(t *testing.T) {
	t.Parallel()

	t.Run("uppercase UUID is normalized to lowercase", func(t *testing.T) {
		t.Parallel()

		useCase, repo, management := devicesTest(t)

		// Input DTO with uppercase GUID
		inputDTO := &dto.Device{
			GUID:     "AAF0C395-C2A2-992E-5655-48210B50D8C9",
			TenantID: "tenant-id-456",
			Tags:     []string{},
		}

		// Expected entity with lowercase GUID (after normalization)
		expectedEntity := &entity.Device{
			ConnectionType: entity.ConnectionTypeDirect,
			GUID:           "aaf0c395-c2a2-992e-5655-48210b50d8c9",
			TenantID:       "tenant-id-456",
			Password:       "encrypted",
		}

		// Expected DTO result
		expectedDTO := &dto.Device{
			ConnectionType: entity.ConnectionTypeDirect,
			GUID:           "aaf0c395-c2a2-992e-5655-48210b50d8c9",
			TenantID:       "tenant-id-456",
			Tags:           nil,
		}

		repo.EXPECT().
			Update(context.Background(), updatedDevice{expectedEntity}).
			Return(true, nil)
		repo.EXPECT().
			GetByID(context.Background(), "aaf0c395-c2a2-992e-5655-48210b50d8c9", "tenant-id-456").
			Return(expectedEntity, nil)
		management.EXPECT().
			DestroyWsmanClient(*expectedDTO)

		result, err := useCase.Update(context.Background(), inputDTO, nil)

		require.NoError(t, err)
		require.Equal(t, "aaf0c395-c2a2-992e-5655-48210b50d8c9", result.GUID)
	})
}

// TestUpdatePartial verifies that when a fields map is supplied, Update
// performs a merge: fields not listed retain their existing (decrypted)
// values and only listed keys are overwritten before persisting.
func TestUpdatePartial(t *testing.T) {
	t.Parallel()

	// Stored entity has encrypted secrets and old hostname.
	existing := &entity.Device{
		GUID:         "device-guid-123",
		TenantID:     "tenant-id-456",
		Hostname:     "old-hostname",
		DeviceInfo:   `{"fwVersion":"11.8.50","fwBuild":"3400","ipAddress":"10.0.0.1","lmsInstalled":false}`,
		Tags:         "lab,floor-2",
		MPSUsername:  "admin",
		Username:     "amtadmin",
		Password:     "encrypted-amt",
		MPSPassword:  ptr("encrypted-mps"),
		MEBXPassword: ptr("encrypted-mebx"),
	}

	firstDiscovered := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	lastSynced := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)

	// Client sends guid, a new hostname, and a partial deviceInfo update.
	discovered := true
	incoming := &dto.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
		Hostname: "new-hostname",
		DeviceInfo: &dto.DeviceInfo{
			FWVersion:       "16.1.30",
			Discovered:      &discovered,
			FirstDiscovered: &firstDiscovered,
			IPAddress:       "10.0.0.55",
			LastSynced:      &lastSynced,
			LMSInstalled:    boolPtr(true),
		},
	}
	fields := map[string]bool{
		"guid":                       true,
		"tenantId":                   true,
		"hostname":                   true,
		"deviceinfo":                 true,
		"deviceinfo.fwversion":       true,
		"deviceinfo.discovered":      true,
		"deviceinfo.firstdiscovered": true,
		"deviceinfo.ipaddress":       true,
		"deviceinfo.lastsynced":      true,
		"deviceinfo.lmsinstalled":    true,
	}

	// After merge + dtoToEntity (MockCrypto re-encrypts plaintext to "encrypted"):
	// Only fields without omitempty tag are included in JSON for zero values.
	expectedDeviceInfoJSON := `{"fwVersion":"16.1.30","fwBuild":"3400","fwSku":"","discovered":true,"firstDiscovered":"2026-05-20T00:00:00Z","currentMode":"","features":"","ipAddress":"10.0.0.55","lastSynced":"2026-05-21T00:00:00Z","lmsInstalled":true}`
	expectedEntity := &entity.Device{
		ConnectionType: entity.ConnectionTypeCIRA,
		GUID:           "device-guid-123",
		TenantID:       "tenant-id-456",
		Hostname:       "new-hostname",
		DeviceInfo:     expectedDeviceInfoJSON,
		Tags:           "lab,floor-2",
		MPSUsername:    "admin",
		Username:       "amtadmin",
		Password:       "encrypted",
		MPSPassword:    ptr("encrypted"),
		MEBXPassword:   ptr("encrypted"),
		Discovered:     boolPtr(true),
	}

	expectedDTO := &dto.Device{
		ConnectionType: entity.ConnectionTypeCIRA,
		GUID:           "device-guid-123",
		TenantID:       "tenant-id-456",
		Hostname:       "new-hostname",
		DeviceInfo: &dto.DeviceInfo{
			FWVersion:       "16.1.30",
			FWBuild:         "3400",
			Discovered:      boolPtr(true),
			FirstDiscovered: &firstDiscovered,
			IPAddress:       "10.0.0.55",
			LastSynced:      &lastSynced,
			LMSInstalled:    boolPtr(true),
		},
		Tags:         []string{"lab", "floor-2"},
		MPSUsername:  "admin",
		Username:     "amtadmin",
		MPSPassword:  "encrypted",
		MEBXPassword: "encrypted",
	}

	t.Run("preserves untouched fields", func(t *testing.T) {
		t.Parallel()

		useCase, repo, management := devicesTest(t)

		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(existing, nil)
		repo.EXPECT().
			Update(context.Background(), gomock.Any()).
			DoAndReturn(func(_ context.Context, actualEntity *entity.Device) (bool, error) {
				require.NotNil(t, actualEntity)
				require.JSONEq(t, expectedDeviceInfoJSON, actualEntity.DeviceInfo)

				// Update stamps a fresh server-managed LastUpdated; assert it was
				// set, then zero it for the field-by-field comparison below.
				require.NotEmpty(t, actualEntity.LastUpdated)

				expectedWithoutInfo := *expectedEntity
				actualWithoutInfo := *actualEntity
				expectedWithoutInfo.DeviceInfo = ""
				actualWithoutInfo.DeviceInfo = ""
				actualWithoutInfo.LastUpdated = ""
				require.Equal(t, expectedWithoutInfo, actualWithoutInfo)

				var actualInfo dto.DeviceInfo
				require.NoError(t, json.Unmarshal([]byte(actualEntity.DeviceInfo), &actualInfo))
				require.Equal(t, "3400", actualInfo.FWBuild)

				return true, nil
			})
		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(expectedEntity, nil)
		management.EXPECT().DestroyWsmanClient(*expectedDTO)

		result, err := useCase.Update(context.Background(), incoming, fields)

		require.NoError(t, err)
		require.Equal(t, expectedDTO, result)
	})

	t.Run("propagates not-found from initial fetch", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(nil, nil)

		result, err := useCase.Update(context.Background(), incoming, fields)

		require.Nil(t, result)
		require.IsType(t, devices.ErrNotFound, err)
	})

	t.Run("propagates database error from initial fetch", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(nil, devices.ErrDatabase)

		result, err := useCase.Update(context.Background(), incoming, fields)

		require.Nil(t, result)
		require.Error(t, err)
		require.Contains(t, err.Error(), devices.ErrDatabase.Error())
	})
}

// TestUpdatePreservesFirstDiscovered verifies firstDiscovered and discovered are
// immutable once set: a client PATCH cannot overwrite the stored values, while
// lastSynced still advances.
func TestUpdatePreservesFirstDiscovered(t *testing.T) {
	t.Parallel()

	original := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)

	// Stored record already discovered on 2026-05-20.
	existing := &entity.Device{
		GUID:       "device-guid-123",
		TenantID:   "tenant-id-456",
		DeviceInfo: `{"discovered":false,"firstDiscovered":"2026-05-20T00:00:00Z","lastSynced":"2026-05-20T00:00:00Z"}`,
		Password:   "encrypted-amt",
	}

	// Client tries to change firstDiscovered/discovered and bumps lastSynced.
	tampered := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	newSync := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	discoveredTrue := true
	incoming := &dto.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
		DeviceInfo: &dto.DeviceInfo{
			Discovered:      &discoveredTrue,
			FirstDiscovered: &tampered,
			LastSynced:      &newSync,
		},
	}
	fields := map[string]bool{
		"deviceinfo":                 true,
		"deviceinfo.discovered":      true,
		"deviceinfo.firstdiscovered": true,
		"deviceinfo.lastsynced":      true,
	}

	useCase, repo, management := devicesTest(t)

	repo.EXPECT().
		GetByID(context.Background(), "device-guid-123", "tenant-id-456").
		Return(existing, nil)
	repo.EXPECT().
		Update(context.Background(), gomock.Any()).
		DoAndReturn(func(_ context.Context, actualEntity *entity.Device) (bool, error) {
			var info dto.DeviceInfo
			require.NoError(t, json.Unmarshal([]byte(actualEntity.DeviceInfo), &info))

			// Immutable fields keep their stored values.
			require.NotNil(t, info.FirstDiscovered)
			require.Equal(t, original, *info.FirstDiscovered)
			require.NotNil(t, info.Discovered)
			require.False(t, *info.Discovered)

			// lastSynced advances.
			require.NotNil(t, info.LastSynced)
			require.Equal(t, newSync, *info.LastSynced)

			return true, nil
		})
	repo.EXPECT().
		GetByID(context.Background(), "device-guid-123", "tenant-id-456").
		Return(existing, nil)
	management.EXPECT().DestroyWsmanClient(gomock.Any())

	_, err := useCase.Update(context.Background(), incoming, fields)
	require.NoError(t, err)
}

// TestUpdatePreservesFirstDiscoveredWholesale covers the backward-compat path where
// the caller sends only a top-level "deviceinfo" key (no nested deviceinfo.* keys),
// which replaces DeviceInfo wholesale. firstDiscovered/discovered must still be kept
// from the stored record.
func TestUpdatePreservesFirstDiscoveredWholesale(t *testing.T) {
	t.Parallel()

	original := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)

	existing := &entity.Device{
		GUID:       "device-guid-123",
		TenantID:   "tenant-id-456",
		DeviceInfo: `{"discovered":false,"firstDiscovered":"2026-05-20T00:00:00Z","lastSynced":"2026-05-20T00:00:00Z"}`,
		Password:   "encrypted-amt",
	}

	tampered := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	newSync := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	discoveredTrue := true
	incoming := &dto.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
		DeviceInfo: &dto.DeviceInfo{
			Discovered:      &discoveredTrue,
			FirstDiscovered: &tampered,
			LastSynced:      &newSync,
		},
	}
	// Only the top-level key — no nested deviceinfo.* keys.
	fields := map[string]bool{"deviceinfo": true}

	useCase, repo, management := devicesTest(t)

	repo.EXPECT().
		GetByID(context.Background(), "device-guid-123", "tenant-id-456").
		Return(existing, nil)
	repo.EXPECT().
		Update(context.Background(), gomock.Any()).
		DoAndReturn(func(_ context.Context, actualEntity *entity.Device) (bool, error) {
			var info dto.DeviceInfo
			require.NoError(t, json.Unmarshal([]byte(actualEntity.DeviceInfo), &info))

			// Immutable fields keep their stored values even on wholesale replace.
			require.NotNil(t, info.FirstDiscovered)
			require.Equal(t, original, *info.FirstDiscovered)
			require.NotNil(t, info.Discovered)
			require.False(t, *info.Discovered)

			// Non-immutable field is taken from the incoming payload.
			require.NotNil(t, info.LastSynced)
			require.Equal(t, newSync, *info.LastSynced)

			return true, nil
		})
	repo.EXPECT().
		GetByID(context.Background(), "device-guid-123", "tenant-id-456").
		Return(existing, nil)
	management.EXPECT().DestroyWsmanClient(gomock.Any())

	_, err := useCase.Update(context.Background(), incoming, fields)
	require.NoError(t, err)
}

func TestUpdateConnectionStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		guid   string
		status bool
		mock   func(*mocks.MockDeviceManagementRepository)
		err    error
	}{
		{
			name:   "successful connection status update - connected",
			guid:   "device-guid-123",
			status: true,
			mock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					UpdateConnectionStatus(context.Background(), "device-guid-123", true).
					Return(nil)
			},
			err: nil,
		},
		{
			name:   "successful connection status update - disconnected",
			guid:   "device-guid-123",
			status: false,
			mock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					UpdateConnectionStatus(context.Background(), "device-guid-123", false).
					Return(nil)
			},
			err: nil,
		},
		{
			name:   "mixed-case GUID is normalized to lowercase",
			guid:   "DEVICE-GUID-123",
			status: true,
			mock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					UpdateConnectionStatus(context.Background(), "device-guid-123", true).
					Return(nil)
			},
			err: nil,
		},
		{
			name:   "database error",
			guid:   "device-guid-123",
			status: true,
			mock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					UpdateConnectionStatus(context.Background(), "device-guid-123", true).
					Return(devices.ErrDatabase)
			},
			err: devices.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, _ := devicesTest(t)

			tc.mock(repo)

			err := useCase.UpdateConnectionStatus(context.Background(), tc.guid, tc.status)

			require.IsType(t, tc.err, err)
		})
	}
}

func TestUpdateLastSeen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		guid string
		mock func(*mocks.MockDeviceManagementRepository)
		err  error
	}{
		{
			name: "successful update",
			guid: "device-guid-123",
			mock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					UpdateLastSeen(context.Background(), "device-guid-123").
					Return(nil)
			},
			err: nil,
		},
		{
			name: "mixed-case GUID is normalized to lowercase",
			guid: "DEVICE-GUID-123",
			mock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					UpdateLastSeen(context.Background(), "device-guid-123").
					Return(nil)
			},
			err: nil,
		},
		{
			name: "database error",
			guid: "device-guid-123",
			mock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					UpdateLastSeen(context.Background(), "device-guid-123").
					Return(devices.ErrDatabase)
			},
			err: devices.ErrDatabase,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, _ := devicesTest(t)

			tc.mock(repo)

			err := useCase.UpdateLastSeen(context.Background(), tc.guid)

			require.IsType(t, tc.err, err)
		})
	}
}

// TestInsert_ConnectionType covers the server-side connectionType rule on
// create: derived from the MPS credentials when absent, canonicalized when
// supplied, rejected when it is neither CIRA nor Direct.
func TestInsert_ConnectionType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   dto.Device
		want string
	}{
		{
			name: "derives Direct without MPS credentials",
			in:   dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456"},
			want: entity.ConnectionTypeDirect,
		},
		{
			name: "derives CIRA from mpsusername",
			in:   dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", MPSUsername: "admin"},
			want: entity.ConnectionTypeCIRA,
		},
		{
			name: "canonicalizes an explicit lowercase value",
			in:   dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", ConnectionType: "cira"},
			want: entity.ConnectionTypeCIRA,
		},
		{
			name: "keeps an explicit value over the derived one",
			in:   dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", MPSUsername: "admin", ConnectionType: "Direct"},
			want: entity.ConnectionTypeDirect,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, _ := devicesTest(t)

			repo.EXPECT().
				Insert(context.Background(), gomock.Any()).
				DoAndReturn(func(_ context.Context, d *entity.Device) (string, error) {
					require.Equal(t, tt.want, d.ConnectionType)

					return "", nil
				})
			repo.EXPECT().
				GetByID(context.Background(), "device-guid-123", "tenant-id-456").
				Return(&entity.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", ConnectionType: tt.want}, nil)

			in := tt.in

			got, err := useCase.Insert(context.Background(), &in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.ConnectionType)
		})
	}

	t.Run("rejects an unknown value before touching the repository", func(t *testing.T) {
		t.Parallel()

		useCase, _, _ := devicesTest(t)

		_, err := useCase.Insert(context.Background(), &dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", ConnectionType: "bogus"})

		var validationErr devices.ValidationError

		require.ErrorAs(t, err, &validationErr)
		require.Contains(t, validationErr.Console.FriendlyMessage(), "connectionType must be CIRA or Direct")
	})
}

// TestUpdate_ConnectionType covers the rule on PATCH: a request that omits
// connectionType re-derives it from the merged MPS credentials (backfilling
// legacy rows), an explicit value is canonicalized, and unknown values fail.
func TestUpdate_ConnectionType(t *testing.T) {
	t.Parallel()

	stored := func() *entity.Device {
		return &entity.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", MPSUsername: "admin", Password: "encrypted"}
	}

	tests := []struct {
		name   string
		in     dto.Device
		fields map[string]bool
		want   string
	}{
		{
			name:   "omitted connectionType is derived from stored mpsusername",
			in:     dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", Hostname: "renamed"},
			fields: map[string]bool{"hostname": true},
			want:   entity.ConnectionTypeCIRA,
		},
		{
			name:   "clearing mpsusername flips the derived value to Direct",
			in:     dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456"},
			fields: map[string]bool{"mpsusername": true},
			want:   entity.ConnectionTypeDirect,
		},
		{
			name:   "explicit value is canonicalized",
			in:     dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", ConnectionType: "direct"},
			fields: map[string]bool{"connectiontype": true},
			want:   entity.ConnectionTypeDirect,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, repo, management := devicesTest(t)

			repo.EXPECT().
				GetByID(context.Background(), "device-guid-123", "tenant-id-456").
				Return(stored(), nil)
			repo.EXPECT().
				Update(context.Background(), gomock.Any()).
				DoAndReturn(func(_ context.Context, d *entity.Device) (bool, error) {
					require.Equal(t, tt.want, d.ConnectionType)

					return true, nil
				})
			repo.EXPECT().
				GetByID(context.Background(), "device-guid-123", "tenant-id-456").
				Return(&entity.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", ConnectionType: tt.want}, nil)
			management.EXPECT().DestroyWsmanClient(gomock.Any())

			in := tt.in

			got, err := useCase.Update(context.Background(), &in, tt.fields)
			require.NoError(t, err)
			require.Equal(t, tt.want, got.ConnectionType)
		})
	}

	t.Run("rejects an unknown value", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(stored(), nil)

		in := dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", ConnectionType: "bogus"}

		_, err := useCase.Update(context.Background(), &in, map[string]bool{"connectiontype": true})

		var validationErr devices.ValidationError

		require.ErrorAs(t, err, &validationErr)
	})
}

// TestGetByID_HidesSoftDeleted: the repository key lookups return flagged rows
// so Insert can restore them, but the API-facing lookups must report not found.
func TestGetByID_HidesSoftDeleted(t *testing.T) {
	t.Parallel()

	deleted := &entity.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", IsDeleted: true, DeletedDate: "2026-06-19T09:30:00Z"}

	t.Run("GetByID", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)
		repo.EXPECT().GetByID(context.Background(), "device-guid-123", "tenant-id-456").Return(deleted, nil)

		_, err := useCase.GetByID(context.Background(), "device-guid-123", "tenant-id-456", false)

		var notFoundErr repoerrors.NotFoundError

		require.ErrorAs(t, err, &notFoundErr)
	})

	t.Run("GetByGUID", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)
		repo.EXPECT().GetByGUID(context.Background(), "device-guid-123").Return(deleted, nil)

		_, err := useCase.GetByGUID(context.Background(), "device-guid-123", false)

		var notFoundErr repoerrors.NotFoundError

		require.ErrorAs(t, err, &notFoundErr)
	})
}

// TestInsert_RestoresSoftDeleted: adding a device whose GUID belongs to a
// soft-deleted record revives that record with the new details, keeping its
// original id and createdDate. A live duplicate still surfaces the conflict.
func TestInsert_RestoresSoftDeleted(t *testing.T) {
	t.Parallel()

	conflict := repoerrors.NotUniqueError{}

	t.Run("revives the deleted record in place", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		existing := &entity.Device{
			GUID:        "device-guid-123",
			TenantID:    "tenant-id-456",
			ID:          "original-id",
			CreatedDate: "2026-01-01T00:00:00Z",
			Hostname:    "old-host",
			IsDeleted:   true,
			DeletedDate: "2026-06-19T09:30:00Z",
		}
		restored := &entity.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", ID: "original-id", CreatedDate: "2026-01-01T00:00:00Z", Hostname: "new-host"}

		repo.EXPECT().Insert(context.Background(), gomock.Any()).Return("", conflict)
		repo.EXPECT().GetByID(context.Background(), "device-guid-123", "tenant-id-456").Return(existing, nil)
		repo.EXPECT().
			Update(context.Background(), gomock.Any()).
			DoAndReturn(func(_ context.Context, d *entity.Device) (bool, error) {
				require.Equal(t, "original-id", d.ID, "restore keeps the original id")
				require.Equal(t, "2026-01-01T00:00:00Z", d.CreatedDate, "restore keeps the original createdDate")
				require.Equal(t, "new-host", d.Hostname, "restore applies the new details")
				require.False(t, d.IsDeleted)
				require.Empty(t, d.DeletedDate)
				require.NotEmpty(t, d.LastUpdated)

				return true, nil
			})
		repo.EXPECT().GetByID(context.Background(), "device-guid-123", "tenant-id-456").Return(restored, nil)

		got, err := useCase.Insert(context.Background(), &dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456", Hostname: "new-host"})
		require.NoError(t, err)
		require.Equal(t, "original-id", got.ID)
		require.Equal(t, "new-host", got.Hostname)
	})

	t.Run("live duplicate still conflicts", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		repo.EXPECT().Insert(context.Background(), gomock.Any()).Return("", conflict)
		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(&entity.Device{GUID: "device-guid-123", TenantID: "tenant-id-456"}, nil)

		// gomock fails the test if Update is called, so a returned error proves
		// the conflict surfaced unchanged instead of being treated as a restore.
		_, err := useCase.Insert(context.Background(), &dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456"})
		require.Error(t, err)
	})

	t.Run("non-conflict errors are not retried", func(t *testing.T) {
		t.Parallel()

		useCase, repo, _ := devicesTest(t)

		repo.EXPECT().Insert(context.Background(), gomock.Any()).Return("", devices.ErrDatabase)

		_, err := useCase.Insert(context.Background(), &dto.Device{GUID: "device-guid-123", TenantID: "tenant-id-456"})
		require.Error(t, err)
	})
}
