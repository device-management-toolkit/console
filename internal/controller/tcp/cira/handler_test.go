package cira

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type mockSecretReader struct {
	secret map[string]string
	err    error
}

func (m mockSecretReader) GetObject(_ string) (map[string]string, error) {
	return m.secret, m.err
}

const (
	testGUID     = "4c4c4544-0043-4810-8053-b8c04f595931"
	testUsername = "mpsuser"
	testPassword = "mpspass"
)

func authHandler(t *testing.T, device *dto.Device, err error) *APFHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
	mockDevices.EXPECT().GetByGUID(gomock.Any(), testGUID, true).Return(device, err)

	handler := NewAPFHandler(mockDevices, logger.New("error"))
	require.NoError(t, handler.OnProtocolVersion(apf.ProtocolVersionInfo{UUID: testGUID}))

	return handler
}

// A device whose row carries a non-empty tenant must still authenticate: CIRA
// devices present only a GUID and MPS credentials.
func TestOnAuthRequestAcceptsDeviceInNonDefaultTenant(t *testing.T) {
	t.Parallel()

	handler := authHandler(t, &dto.Device{
		GUID:        testGUID,
		MPSUsername: testUsername,
		MPSPassword: testPassword,
		TenantID:    "acme-corp",
	}, nil)

	response := handler.OnAuthRequest(apf.AuthRequest{
		MethodName: "password",
		Username:   testUsername,
		Password:   testPassword,
	})

	require.True(t, response.Authenticated)
}

func TestOnAuthRequestLearnsDefaultTenant(t *testing.T) {
	t.Parallel()

	handler := authHandler(t, &dto.Device{
		GUID:        testGUID,
		MPSUsername: testUsername,
		MPSPassword: testPassword,
	}, nil)

	response := handler.OnAuthRequest(apf.AuthRequest{
		MethodName: "password",
		Username:   testUsername,
		Password:   testPassword,
	})

	require.True(t, response.Authenticated)
}

func TestOnAuthRequestUsesVaultPasswordWhenDeviceRowHasNone(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
	mockDevices.EXPECT().GetByGUID(gomock.Any(), testGUID, true).Return(&dto.Device{
		GUID:        testGUID,
		MPSUsername: testUsername,
	}, nil)

	handler := NewAPFHandler(mockDevices, logger.New("error"), mockSecretReader{
		secret: map[string]string{secretKeyMPSPassword: testPassword},
	})
	require.NoError(t, handler.OnProtocolVersion(apf.ProtocolVersionInfo{UUID: testGUID}))

	response := handler.OnAuthRequest(apf.AuthRequest{
		MethodName: "password",
		Username:   testUsername,
		Password:   testPassword,
	})

	require.True(t, response.Authenticated)
}

func TestOnAuthRequestRejectsWhenVaultCredentialReadFails(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)
	mockDevices.EXPECT().GetByGUID(gomock.Any(), testGUID, true).Return(&dto.Device{
		GUID:        testGUID,
		MPSUsername: testUsername,
	}, nil)

	handler := NewAPFHandler(mockDevices, logger.New("error"), mockSecretReader{err: errors.New("vault unavailable")})
	require.NoError(t, handler.OnProtocolVersion(apf.ProtocolVersionInfo{UUID: testGUID}))

	response := handler.OnAuthRequest(apf.AuthRequest{
		MethodName: "password",
		Username:   testUsername,
		Password:   testPassword,
	})

	require.False(t, response.Authenticated)
}

func TestOnAuthRequestRejectsBadPasswordWithoutLearningTenant(t *testing.T) {
	t.Parallel()

	handler := authHandler(t, &dto.Device{
		GUID:        testGUID,
		MPSUsername: testUsername,
		MPSPassword: testPassword,
		TenantID:    "acme-corp",
	}, nil)

	response := handler.OnAuthRequest(apf.AuthRequest{
		MethodName: "password",
		Username:   testUsername,
		Password:   "wrong",
	})

	require.False(t, response.Authenticated)
}

func TestOnAuthRequestRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)

	handler := NewAPFHandler(mockDevices, logger.New("error"))
	require.NoError(t, handler.OnProtocolVersion(apf.ProtocolVersionInfo{UUID: testGUID}))

	response := handler.OnAuthRequest(apf.AuthRequest{MethodName: "publickey"})

	require.False(t, response.Authenticated)
}
