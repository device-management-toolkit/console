package main

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(name string, arg ...string) error {
	args := m.Called(name, arg)

	return args.Error(0)
}

func TestMainFunction(_ *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	os.Setenv("GIN_MODE", "debug")

	// Mock functions
	initializeConfigFunc = func() (*config.Config, error) {
		return &config.Config{HTTP: config.HTTP{Port: "8080"}, App: config.App{EncryptionKey: "test"}}, nil
	}

	initializeAppFunc = func(_ *config.Config) error {
		return nil
	}

	runAppFunc = func(_ *config.Config) {}

	// Mock certificate functions
	loadOrGenerateRootCertFunc = func(_ security.Storager, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	loadOrGenerateWebServerCertFunc = func(_ security.Storager, _ certificates.CertAndKeyType, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	// Call the main function
	main()
}

func TestOpenBrowserWindows(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "cmd", []string{"/c", "start", "http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "windows")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

func TestOpenBrowserDarwin(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "open", []string{"http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "darwin")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

func TestOpenBrowserLinux(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying executor.
	mockCmdExecutor := new(MockCommandExecutor)
	cmdExecutor = mockCmdExecutor

	mockCmdExecutor.On("Execute", "xdg-open", []string{"http://localhost:8080"}).Return(nil)

	err := openBrowser("http://localhost:8080", "ubuntu")
	assert.NoError(t, err)
	mockCmdExecutor.AssertExpectations(t)
}

type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) GenerateSpec() ([]byte, error) {
	args := m.Called()

	var b []byte

	if v := args.Get(0); v != nil {
		if bb, ok := v.([]byte); ok {
			b = bb
		}
	}

	return b, args.Error(1)
}

func (m *MockGenerator) SaveSpec(b []byte, path string) error {
	args := m.Called(b, path)

	return args.Error(0)
}

//nolint:paralleltest // modifies package-level NewGeneratorFunc
func TestHandleOpenAPIGeneration_Success(t *testing.T) {
	mockGen := new(MockGenerator)

	NewGeneratorFunc = func(_ usecase.Usecases, _ logger.Interface) interface {
		GenerateSpec() ([]byte, error)
		SaveSpec([]byte, string) error
	} {
		return mockGen
	}

	expectedSpec := []byte("{}")
	mockGen.On("GenerateSpec").Return(expectedSpec, nil)
	mockGen.On("SaveSpec", expectedSpec, "doc/openapi.json").Return(nil)

	err := handleOpenAPIGeneration()
	assert.NoError(t, err)

	mockGen.AssertExpectations(t)
}

//nolint:paralleltest // modifies package-level NewGeneratorFunc
func TestHandleOpenAPIGeneration_GenerateFails(t *testing.T) {
	mockGen := new(MockGenerator)

	NewGeneratorFunc = func(_ usecase.Usecases, _ logger.Interface) interface {
		GenerateSpec() ([]byte, error)
		SaveSpec([]byte, string) error
	} {
		return mockGen
	}

	mockGen.On("GenerateSpec").Return([]byte(nil), assert.AnError)

	err := handleOpenAPIGeneration()
	assert.Error(t, err)

	mockGen.AssertExpectations(t)
}

// TestGenerateRandomPassword tests the password generation function.
func TestGenerateRandomPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		length int
	}{
		{"length 8", 8},
		{"length 16", 16},
		{"length 32", 32},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			password, err := generateRandomPassword(tc.length)
			require.NoError(t, err)
			assert.Len(t, password, tc.length)
		})
	}
}

// TestGenerateRandomPassword_Uniqueness ensures generated passwords are unique.
func TestGenerateRandomPassword_Uniqueness(t *testing.T) {
	t.Parallel()

	passwords := make(map[string]bool)

	for i := 0; i < 100; i++ {
		password, err := generateRandomPassword(16)
		require.NoError(t, err)
		assert.False(t, passwords[password], "generated duplicate password")

		passwords[password] = true
	}
}

// TestTryRemoteAdminPassword_NilStorage tests when remote storage is nil.
func TestTryRemoteAdminPassword_NilStorage(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	result := tryRemoteAdminPassword(cfg, nil)

	assert.False(t, result)
	assert.Empty(t, cfg.AdminPassword)
}

// TestTryRemoteAdminPassword_Success tests successful password retrieval from remote.
func TestTryRemoteAdminPassword_Success(t *testing.T) {
	t.Parallel()

	mockStorage := new(mocks.MockStorager)
	mockStorage.On("GetKeyValue", "admin-password").Return("remote-password", nil)

	cfg := &config.Config{}
	result := tryRemoteAdminPassword(cfg, mockStorage)

	assert.True(t, result)
	assert.Equal(t, "remote-password", cfg.AdminPassword)
	mockStorage.AssertExpectations(t)
}

// TestTryRemoteAdminPassword_NotFound tests when password is not in remote storage.
func TestTryRemoteAdminPassword_NotFound(t *testing.T) {
	t.Parallel()

	mockStorage := new(mocks.MockStorager)
	mockStorage.On("GetKeyValue", "admin-password").Return("", security.ErrKeyNotFound)

	cfg := &config.Config{}
	result := tryRemoteAdminPassword(cfg, mockStorage)

	assert.False(t, result)
	assert.Empty(t, cfg.AdminPassword)
	mockStorage.AssertExpectations(t)
}

// TestTryRemoteAdminPassword_EmptyValue tests when remote returns empty value.
func TestTryRemoteAdminPassword_EmptyValue(t *testing.T) {
	t.Parallel()

	mockStorage := new(mocks.MockStorager)
	mockStorage.On("GetKeyValue", "admin-password").Return("", nil)

	cfg := &config.Config{}
	result := tryRemoteAdminPassword(cfg, mockStorage)

	assert.False(t, result)
	assert.Empty(t, cfg.AdminPassword)
	mockStorage.AssertExpectations(t)
}

// TestTryLocalAdminPassword_Success tests successful password retrieval from keyring.
func TestTryLocalAdminPassword_Success(t *testing.T) {
	t.Parallel()

	mockLocal := new(mocks.MockStorager)
	mockLocal.On("GetKeyValue", "admin-password").Return("local-password", nil)

	cfg := &config.Config{}
	result := tryLocalAdminPassword(cfg, mockLocal, nil)

	assert.True(t, result)
	assert.Equal(t, "local-password", cfg.AdminPassword)
	mockLocal.AssertExpectations(t)
}

// TestTryLocalAdminPassword_SuccessWithSync tests retrieval from local with sync to remote.
func TestTryLocalAdminPassword_SuccessWithSync(t *testing.T) {
	t.Parallel()

	mockLocal := new(mocks.MockStorager)
	mockRemote := new(mocks.MockStorager)

	mockLocal.On("GetKeyValue", "admin-password").Return("local-password", nil)
	mockRemote.On("SetKeyValue", "admin-password", "local-password").Return(nil)

	cfg := &config.Config{}
	result := tryLocalAdminPassword(cfg, mockLocal, mockRemote)

	assert.True(t, result)
	assert.Equal(t, "local-password", cfg.AdminPassword)
	mockLocal.AssertExpectations(t)
	mockRemote.AssertExpectations(t)
}

// TestTryLocalAdminPassword_NotFound tests when password is not in local keyring.
func TestTryLocalAdminPassword_NotFound(t *testing.T) {
	t.Parallel()

	mockLocal := new(mocks.MockStorager)
	mockLocal.On("GetKeyValue", "admin-password").Return("", security.ErrKeyNotFound)

	cfg := &config.Config{}
	result := tryLocalAdminPassword(cfg, mockLocal, nil)

	assert.False(t, result)
	assert.Empty(t, cfg.AdminPassword)
	mockLocal.AssertExpectations(t)
}

// TestTryLocalAdminPassword_EmptyValue tests when local returns empty value.
func TestTryLocalAdminPassword_EmptyValue(t *testing.T) {
	t.Parallel()

	mockLocal := new(mocks.MockStorager)
	mockLocal.On("GetKeyValue", "admin-password").Return("", nil)

	cfg := &config.Config{}
	result := tryLocalAdminPassword(cfg, mockLocal, nil)

	assert.False(t, result)
	assert.Empty(t, cfg.AdminPassword)
	mockLocal.AssertExpectations(t)
}

// TestSyncAdminPasswordToRemote_NilStorage tests sync when remote is nil.
func TestSyncAdminPasswordToRemote_NilStorage(t *testing.T) {
	t.Parallel()

	// Should not panic when remote is nil
	syncAdminPasswordToRemote("password", nil)
}

// TestSyncAdminPasswordToRemote_Success tests successful sync to remote.
func TestSyncAdminPasswordToRemote_Success(t *testing.T) {
	t.Parallel()

	mockRemote := new(mocks.MockStorager)
	mockRemote.On("SetKeyValue", "admin-password", "test-password").Return(nil)

	syncAdminPasswordToRemote("test-password", mockRemote)

	mockRemote.AssertExpectations(t)
}

// TestSyncAdminPasswordToRemote_Error tests sync failure handling.
func TestSyncAdminPasswordToRemote_Error(t *testing.T) {
	t.Parallel()

	mockRemote := new(mocks.MockStorager)
	mockRemote.On("SetKeyValue", "admin-password", "test-password").Return(errors.New("sync failed"))

	// Should not panic on error, just log warning
	syncAdminPasswordToRemote("test-password", mockRemote)

	mockRemote.AssertExpectations(t)
}

// TestSaveAdminPassword_RemoteSuccess tests saving to remote storage.
func TestSaveAdminPassword_RemoteSuccess(t *testing.T) {
	t.Parallel()

	mockRemote := new(mocks.MockStorager)
	mockLocal := new(mocks.MockStorager)

	mockRemote.On("SetKeyValue", "admin-password", "test-password").Return(nil)

	err := saveAdminPassword("test-password", mockRemote, mockLocal)

	assert.NoError(t, err)
	mockRemote.AssertExpectations(t)
	mockLocal.AssertNotCalled(t, "SetKeyValue")
}

// TestSaveAdminPassword_RemoteError tests fallback behavior is not attempted when remote fails.
func TestSaveAdminPassword_RemoteError(t *testing.T) {
	t.Parallel()

	mockRemote := new(mocks.MockStorager)
	mockLocal := new(mocks.MockStorager)

	mockRemote.On("SetKeyValue", "admin-password", "test-password").Return(errors.New("remote error"))

	err := saveAdminPassword("test-password", mockRemote, mockLocal)

	assert.Error(t, err)
	mockRemote.AssertExpectations(t)
	// Local should not be called when remote is configured but fails
	mockLocal.AssertNotCalled(t, "SetKeyValue")
}

// TestSaveAdminPassword_LocalSuccess tests saving to local keyring when remote is nil.
func TestSaveAdminPassword_LocalSuccess(t *testing.T) {
	t.Parallel()

	mockLocal := new(mocks.MockStorager)
	mockLocal.On("SetKeyValue", "admin-password", "test-password").Return(nil)

	err := saveAdminPassword("test-password", nil, mockLocal)

	assert.NoError(t, err)
	mockLocal.AssertExpectations(t)
}

// TestSaveAdminPassword_LocalError tests local save failure.
func TestSaveAdminPassword_LocalError(t *testing.T) {
	t.Parallel()

	mockLocal := new(mocks.MockStorager)
	mockLocal.On("SetKeyValue", "admin-password", "test-password").Return(errors.New("local error"))

	err := saveAdminPassword("test-password", nil, mockLocal)

	assert.Error(t, err)
	mockLocal.AssertExpectations(t)
}

// TestHandleAdminPassword_AlreadyConfigured tests when password is already set.
func TestHandleAdminPassword_AlreadyConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Auth: config.Auth{
			AdminPassword: "already-set",
		},
	}

	handleAdminPassword(cfg)

	assert.Equal(t, "already-set", cfg.AdminPassword)
}
