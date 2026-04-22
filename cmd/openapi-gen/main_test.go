package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

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

func installMockGenerator(t *testing.T) *MockGenerator {
	t.Helper()

	mockGen := new(MockGenerator)
	originalNewGenerator := NewGeneratorFunc

	NewGeneratorFunc = func(_ usecase.Usecases, _ logger.Interface) interface {
		GenerateSpec() ([]byte, error)
		SaveSpec([]byte, string) error
	} {
		return mockGen
	}

	t.Cleanup(func() { NewGeneratorFunc = originalNewGenerator })

	return mockGen
}

//nolint:paralleltest // modifies package-level NewGeneratorFunc
func TestGenerate_Success(t *testing.T) {
	mockGen := installMockGenerator(t)

	expectedSpec := []byte("{}")
	mockGen.On("GenerateSpec").Return(expectedSpec, nil)
	mockGen.On("SaveSpec", expectedSpec, outputPath).Return(nil)

	err := generate(logger.New("info"))

	assert.NoError(t, err)
	mockGen.AssertExpectations(t)
}

//nolint:paralleltest // modifies package-level NewGeneratorFunc
func TestGenerate_GenerateFails(t *testing.T) {
	mockGen := installMockGenerator(t)

	mockGen.On("GenerateSpec").Return([]byte(nil), assert.AnError)

	err := generate(logger.New("info"))

	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	mockGen.AssertExpectations(t)
}

//nolint:paralleltest // modifies package-level NewGeneratorFunc
func TestGenerate_SaveFails(t *testing.T) {
	mockGen := installMockGenerator(t)

	expectedSpec := []byte("{}")
	mockGen.On("GenerateSpec").Return(expectedSpec, nil)
	mockGen.On("SaveSpec", expectedSpec, outputPath).Return(assert.AnError)

	err := generate(logger.New("info"))

	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	mockGen.AssertExpectations(t)
}
