package mocks

import "github.com/stretchr/testify/mock"

// MockStorager implements security.Storager for testing.
type MockStorager struct {
	mock.Mock
}

func (m *MockStorager) GetKeyValue(key string) (string, error) {
	args := m.Called(key)

	return args.String(0), args.Error(1)
}

func (m *MockStorager) SetKeyValue(key, value string) error {
	args := m.Called(key, value)

	return args.Error(0)
}

func (m *MockStorager) DeleteKeyValue(key string) error {
	args := m.Called(key)

	return args.Error(0)
}
