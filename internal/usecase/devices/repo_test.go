package devices_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func ptr(s string) *string {
	return &s
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
				require.Contains(t, err.Error(), tc.err.Error())
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
					Delete(context.Background(), "guid-123", "tenant-id-456").
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
					Delete(context.Background(), "guid-456", "tenant-id-456").
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
				require.Equal(t, err.Error(), tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:         "device-guid-123",
		TenantID:     "tenant-id-456",
		Password:     "encrypted",
		MPSPassword:  nil,
		MEBXPassword: nil,
		Tags:         "hello,test",
	}

	deviceDTO := &dto.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
		Tags:     []string{"hello", "test"},
	}

	tests := []testUsecase{
		{
			name: "successful update",
			mock: func(repo *mocks.MockDeviceManagementRepository, management *mocks.MockWSMAN) {
				repo.EXPECT().
					Update(context.Background(), device).
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
					Update(context.Background(), device).
					Return(false, nil)
			},
			res: (*dto.Device)(nil),
			err: devices.ErrNotFound,
		},
		{
			name: "update fails - database error",
			mock: func(repo *mocks.MockDeviceManagementRepository, _ *mocks.MockWSMAN) {
				repo.EXPECT().
					Update(context.Background(), device).
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

			result, err := useCase.Update(context.Background(), deviceDTO)

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
					GUID:         "device-guid-123",
					Password:     "encrypted",
					MPSPassword:  nil,
					MEBXPassword: nil,
					TenantID:     "tenant-id-456",
				}

				repo.EXPECT().
					Insert(context.Background(), device).
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
					GUID:         "device-guid-123",
					Password:     "encrypted",
					MPSPassword:  nil,
					MEBXPassword: nil,
					TenantID:     "tenant-id-456",
				}

				repo.EXPECT().
					Insert(context.Background(), device).
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
		GUID:         "device-guid-123",
		TenantID:     "tenant-id-456",
		Password:     "encrypted",
		MPSPassword:  ptr("encrypted"),
		MEBXPassword: ptr("encrypted"),
		Tags:         "hello,test",
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
		GUID:         "device-guid-123",
		TenantID:     "tenant-id-456",
		Tags:         []string{"hello", "test"},
		MPSPassword:  "encrypted",
		MEBXPassword: "encrypted",
	}

	t.Run("successful update with passwords", func(t *testing.T) {
		t.Parallel()

		useCase, repo, management := devicesTest(t)

		repo.EXPECT().
			Update(context.Background(), deviceWithPasswords).
			Return(true, nil)
		repo.EXPECT().
			GetByID(context.Background(), "device-guid-123", "tenant-id-456").
			Return(deviceWithPasswords, nil)
		management.EXPECT().
			DestroyWsmanClient(*expectedDTO)

		result, err := useCase.Update(context.Background(), deviceDTOWithPasswords)

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
			GUID:         "device-guid-123",
			TenantID:     "tenant-id-456",
			Password:     "encrypted",
			MPSPassword:  ptr("encrypted"),
			MEBXPassword: ptr("encrypted"),
		}

		repo.EXPECT().
			Insert(context.Background(), deviceWithPasswords).
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
				Delete(context.Background(), tc.expectGUID, "tenant-id-456").
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
			GUID:     "aaf0c395-c2a2-992e-5655-48210b50d8c9",
			TenantID: "tenant-id-456",
			Password: "encrypted",
		}

		// Expected DTO result
		expectedDTO := &dto.Device{
			GUID:     "aaf0c395-c2a2-992e-5655-48210b50d8c9",
			TenantID: "tenant-id-456",
			Tags:     nil,
		}

		repo.EXPECT().
			Update(context.Background(), expectedEntity).
			Return(true, nil)
		repo.EXPECT().
			GetByID(context.Background(), "aaf0c395-c2a2-992e-5655-48210b50d8c9", "tenant-id-456").
			Return(expectedEntity, nil)
		management.EXPECT().
			DestroyWsmanClient(*expectedDTO)

		result, err := useCase.Update(context.Background(), inputDTO)

		require.NoError(t, err)
		require.Equal(t, "aaf0c395-c2a2-992e-5655-48210b50d8c9", result.GUID)
	})
}
