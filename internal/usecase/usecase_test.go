package usecase

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/ciraconfigs"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/internal/usecase/profilewificonfigs"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/internal/usecase/wificonfigs"
	"github.com/device-management-toolkit/console/pkg/db"
)

type usecaseTest struct {
	name           string
	initializeFunc func() *Usecases
	expectedResult *Usecases
}

var once sync.Once

func setupConfig() {
	once.Do(func() {
		config.ConsoleConfig = &config.Config{
			App: config.App{
				EncryptionKey: "test",
			},
		}
	})
}

func TestUsecases(t *testing.T) {
	t.Parallel()

	safeRequirements := security.Crypto{
		EncryptionKey: "test",
	}

	tests := []usecaseTest{
		{
			name: "NewUseCases initializes correctly",
			initializeFunc: func() *Usecases {
				mockDB := mocks.NewMockSQLDB()
				mockLogger := mocks.NewMockLogger(nil)

				setupConfig()

				return NewUseCases(mockDB, mockLogger)
			},
			expectedResult: &Usecases{
				Domains: domains.New(sqldb.NewDomainRepo(&db.SQL{}, mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil), safeRequirements),
				Devices: devices.New(sqldb.NewDeviceRepo(&db.SQL{}, mocks.NewMockLogger(nil)), wsman.NewGoWSMANMessages(mocks.NewMockLogger(nil), safeRequirements), devices.NewRedirector(safeRequirements), mocks.NewMockLogger(nil), safeRequirements),
				Profiles: profiles.New(
					sqldb.NewProfileRepo(&db.SQL{}, mocks.NewMockLogger(nil)),
					sqldb.NewWirelessRepo(&db.SQL{}, mocks.NewMockLogger(nil)),
					profilewificonfigs.New(sqldb.NewProfileWiFiConfigsRepo(&db.SQL{}, mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil)),
					ieee8021xconfigs.New(sqldb.NewIEEE8021xRepo(&db.SQL{}, mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil),
					sqldb.NewDomainRepo(&db.SQL{}, mocks.NewMockLogger(nil)),
					sqldb.NewCIRARepo(&db.SQL{}, mocks.NewMockLogger(nil)),
					safeRequirements,
				),
				IEEE8021xProfiles:  ieee8021xconfigs.New(sqldb.NewIEEE8021xRepo(&db.SQL{}, mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil)),
				CIRAConfigs:        ciraconfigs.New(sqldb.NewCIRARepo(&db.SQL{}, mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil), safeRequirements),
				WirelessProfiles:   wificonfigs.New(sqldb.NewWirelessRepo(&db.SQL{}, mocks.NewMockLogger(nil)), ieee8021xconfigs.New(sqldb.NewIEEE8021xRepo(&db.SQL{}, mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil), safeRequirements),
				ProfileWiFiConfigs: profilewificonfigs.New(sqldb.NewProfileWiFiConfigsRepo(&db.SQL{}, mocks.NewMockLogger(nil)), mocks.NewMockLogger(nil)),
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc := tc.initializeFunc()

			require.NotNil(t, uc)
			assert.NotNil(t, uc.Devices)
			assert.NotNil(t, uc.Domains)
			assert.NotNil(t, uc.Profiles)
			assert.NotNil(t, uc.ProfileWiFiConfigs)
			assert.NotNil(t, uc.IEEE8021xProfiles)
			assert.NotNil(t, uc.CIRAConfigs)
			assert.NotNil(t, uc.WirelessProfiles)

			assert.Equal(t, tc.expectedResult.Domains, uc.Domains)
			assert.Equal(t, tc.expectedResult.Devices, uc.Devices)
			assert.Equal(t, tc.expectedResult.Profiles, uc.Profiles)
			assert.Equal(t, tc.expectedResult.ProfileWiFiConfigs, uc.ProfileWiFiConfigs)
			assert.Equal(t, tc.expectedResult.IEEE8021xProfiles, uc.IEEE8021xProfiles)
			assert.Equal(t, tc.expectedResult.CIRAConfigs, uc.CIRAConfigs)
			assert.Equal(t, tc.expectedResult.WirelessProfiles, uc.WirelessProfiles)
		})
	}
}

func TestInitialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level         string
		expectedLevel string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"invalid", "info"},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.level, func(t *testing.T) {
			t.Parallel()
			setupConfig()

			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			mockDB := mocks.NewMockSQLDB()

			mockLogger := mocks.NewMockLogger(mockCtl)

			uc := NewUseCases(mockDB, mockLogger)

			require.NotNil(t, uc)
			assert.NotNil(t, uc.Devices)
		})
	}
}
