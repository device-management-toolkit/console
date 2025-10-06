package devices_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	amtAlarmClock "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/alarmclock"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/alarmclock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func initAlarmsTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
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

func TestGetAlarmOccurrences(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(*device, false, true).
					Return(hmm)
				hmm.EXPECT().
					GetAlarmOccurrences().
					Return([]alarmclock.AlarmClockOccurrence{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: []dto.AlarmClockOccurrence{},
			err: nil,
		},
		{
			name:   "GetById fails",
			action: 0,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: []dto.AlarmClockOccurrence(nil),
			err: devices.ErrGeneral,
		},
		{
			name:   "GetAlarmOccurrences fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetAlarmOccurrences().
					Return([]alarmclock.AlarmClockOccurrence{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: []dto.AlarmClockOccurrence(nil),
			err: ErrGeneral,
		},
		{
			name:   "GetAlarmOccurrences returns nil",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(*device, false, true).
					Return(hmm)
				hmm.EXPECT().
					GetAlarmOccurrences().
					Return(nil, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: []dto.AlarmClockOccurrence{},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initAlarmsTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			tc.repoMock(repo)

			res, err := useCase.GetAlarmOccurrences(context.Background(), device.GUID)

			assert.Equal(t, tc.res, res)

			require.IsType(t, tc.err, err)
		})
	}
}

func TestCreateAlarmOccurrences(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	occ := dto.AlarmClockOccurrenceInput{
		ElementName:        "test",
		InstanceID:         "test",
		StartTime:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Interval:           1,
		DeleteOnCompletion: true,
	}

	tests := []struct {
		name string

		action int

		manMock func(man *mocks.MockWSMAN, man2 *mocks.MockManagement)

		repoMock func(repo *mocks.MockDeviceManagementRepository)

		res dto.AddAlarmOutput

		err error
	}{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(*device, false, true).
					Return(man2)
				man2.EXPECT().
					CreateAlarmOccurrences(occ.InstanceID, occ.StartTime, 1, occ.DeleteOnCompletion).
					Return(amtAlarmClock.AddAlarmOutput{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.AddAlarmOutput{},
			err: nil,
		},
		{
			name:   "GetByID fails",
			action: 0,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: dto.AddAlarmOutput{},
			err: ErrGeneral,
		},
		{
			name:   "CreateAlarmOccurrences fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(*device, false, true).
					Return(man2)
				man2.EXPECT().
					CreateAlarmOccurrences(occ.InstanceID, occ.StartTime, 1, occ.DeleteOnCompletion).
					Return(amtAlarmClock.AddAlarmOutput{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.AddAlarmOutput{},
			err: devices.ErrAMT.Wrap("CreateAlarmOccurrences", "device.CreateAlarmOccurrences", ErrGeneral),
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initAlarmsTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			tc.repoMock(repo)

			res, err := useCase.CreateAlarmOccurrences(context.Background(), device.GUID, occ)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestDeleteAlarmOccurrences(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []struct {
		name     string
		action   int
		manMock  func(man *mocks.MockWSMAN, man2 *mocks.MockManagement)
		repoMock func(repo *mocks.MockDeviceManagementRepository)
		err      error
	}{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(*device, false, true).
					Return(man2)
				man2.EXPECT().
					DeleteAlarmOccurrences("").
					Return(nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: nil,
		},
		{
			name:   "GetById fails",
			action: 0,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			err: ErrGeneral,
		},
		{
			name:   "DeleteAlarmOccurrences fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(*device, false, true).
					Return(man2)
				man2.EXPECT().
					DeleteAlarmOccurrences("").
					Return(ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initAlarmsTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			tc.repoMock(repo)

			err := useCase.DeleteAlarmOccurrences(context.Background(), device.GUID, "")

			if tc.err != nil {
				assert.Equal(t, tc.err, err)
			}
		})
	}
}

func TestParseInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		interval string
		expected int
		err      error
	}{
		{"Parse days only", "P2D", 2880, nil},
		{"Parse hours only", "PT5H", 300, nil},
		{"Parse minutes only", "PT30M", 30, nil},
		{"Parse complex interval", "P1DT6H30M", 1830, nil},
		{"Parse with seconds (ignored)", "P1DT6H30M45S", 1830, nil},
		{"Empty string", "", 0, nil},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := devices.ParseInterval(tc.interval)

			if tc.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
