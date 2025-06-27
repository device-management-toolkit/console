package devices

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var ErrGeneralWsman = errors.New("general error")

type MockClient struct {
	mock.Mock
}

func (m *MockClient) Connect() error {
	args := m.Called()

	return args.Error(0)
}

func (m *MockClient) Send(data []byte) error {
	args := m.Called(data)

	return args.Error(0)
}

func (m *MockClient) Receive() ([]byte, error) {
	args := m.Called()

	return args.Get(0).([]byte), args.Error(1) //nolint:errcheck // It's a test...
}

func (m *MockClient) CloseConnection() error {
	args := m.Called()

	return args.Error(0)
}

func (m *MockClient) Post(msg string) ([]byte, error) {
	args := m.Called(msg)

	return args.Get(0).([]byte), args.Error(1) //nolint:errcheck // It's a test...
}

func (m *MockClient) Listen() ([]byte, error) {
	args := m.Called()

	return args.Get(0).([]byte), args.Error(1) //nolint:errcheck // It's a test...
}

func (m *MockClient) IsAuthenticated() bool {
	args := m.Called()

	return args.Get(0).(bool) //nolint:errcheck // It's a test...
}

func (m *MockClient) GetServerCertificate() (*tls.Certificate, error) {
	args := m.Called()

	return args.Get(0).(*tls.Certificate), args.Error(1) //nolint:errcheck // It's a test...
}

type wsmanTest struct {
	name string
	err  error
}

func TestRedirectConnect(t *testing.T) {
	t.Parallel()

	tests := []wsmanTest{
		{
			name: "Successful Connection",
			err:  nil,
		},

		{
			name: "Connection Error",
			err:  ErrGeneralWsman,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := new(MockClient)
			mockClient.On("Connect").Return(tt.err)
			deviceConnection := &DeviceConnection{
				wsmanMessages: wsman.Messages{
					Client: mockClient,
					AMT:    amt.Messages{},
					CIM:    cim.Messages{},
					IPS:    ips.Messages{},
				},
			}

			redirector := NewRedirector(security.Crypto{})
			err := redirector.RedirectConnect(context.Background(), deviceConnection)
			require.IsType(t, tt.err, err)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedirectSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sendData []byte
		err      error
	}{
		{
			name:     "Successful Send",
			sendData: []byte("test data"),
			err:      nil,
		},
		{
			name:     "Send Error",
			sendData: []byte("test data"),
			err:      ErrGeneralWsman,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new instance of MockClient
			mockClient := new(MockClient)
			// Setup expectations
			mockClient.On("Send", tt.sendData).Return(tt.err)
			// Create a DeviceConnection with the mock client
			deviceConnection := &DeviceConnection{
				wsmanMessages: wsman.Messages{
					Client: mockClient,
					AMT:    amt.Messages{},
					CIM:    cim.Messages{},
					IPS:    ips.Messages{},
				},
			}

			// Create a Redirector instance
			redirector := NewRedirector(security.Crypto{})
			// Call the method under test
			err := redirector.RedirectSend(context.Background(), deviceConnection, tt.sendData)
			// Assert the expected results
			require.Equal(t, tt.err, err)
			// Assert that the mock expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedirectListen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		res  []byte
		err  error
	}{
		{
			name: "Successful Listen",
			res:  []byte("test data"),
			err:  nil,
		},

		{
			name: "Listen Error",
			res:  nil,
			err:  ErrGeneralWsman,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new instance of MockClient
			mockClient := new(MockClient)
			// Setup expectations
			mockClient.On("Receive").Return(tt.res, tt.err)
			// Create a DeviceConnection with the mock client
			deviceConnection := &DeviceConnection{
				wsmanMessages: wsman.Messages{
					Client: mockClient,
					AMT:    amt.Messages{},
					CIM:    cim.Messages{},
					IPS:    ips.Messages{},
				},
			}

			// Create a Redirector instance
			redirector := NewRedirector(security.Crypto{})
			// Call the method under test
			data, err := redirector.RedirectListen(context.Background(), deviceConnection)
			// Assert the expected results
			require.Equal(t, tt.res, data)
			require.Equal(t, tt.err, err)
			// Assert that the mock expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

func TestRedirectClose(t *testing.T) {
	t.Parallel()

	tests := []wsmanTest{
		{
			name: "Successful Close",
			err:  nil,
		},
		{
			name: "Close Error",
			err:  ErrGeneralWsman,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new instance of MockClient
			mockClient := new(MockClient)
			// Setup expectations
			mockClient.On("CloseConnection").Return(tt.err)
			// Create a DeviceConnection with the mock client
			deviceConnection := &DeviceConnection{
				wsmanMessages: wsman.Messages{
					Client: mockClient,
					AMT:    amt.Messages{},
					CIM:    cim.Messages{},
					IPS:    ips.Messages{},
				},
			}
			// Create a Redirector instance
			redirector := NewRedirector(security.Crypto{})
			// Call the method under test
			err := redirector.RedirectClose(context.Background(), deviceConnection)
			// Assert the expected results
			require.Equal(t, tt.err, err)
			// Assert that the mock expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}
