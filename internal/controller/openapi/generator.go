package openapi

import (
	"encoding/json"
	"os"

	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Generator handles OpenAPI specification generation.
type Generator struct {
	usecases usecase.Usecases
	logger   logger.Interface
}

// NewGenerator creates a new OpenAPI generator.
func NewGenerator(usecases usecase.Usecases, log logger.Interface) *Generator {
	return &Generator{
		usecases: usecases,
		logger:   log,
	}
}

// GenerateSpec generates OpenAPI 3.1.0 specification with compliance fixes.
func (g *Generator) GenerateSpec() ([]byte, error) {
	adapter := NewFuegoAdapter(g.usecases, g.logger)
	adapter.RegisterRoutes()

	spec, err := adapter.GetOpenAPISpec()
	if err != nil {
		return nil, err
	}

	var specJSON map[string]interface{}
	if err := json.Unmarshal(spec, &specJSON); err != nil {
		return nil, err
	}

	finalSpec, err := json.MarshalIndent(specJSON, "", "  ")
	if err != nil {
		return nil, err
	}

	return finalSpec, nil
}

// SaveSpec saves the OpenAPI specification to a file using restrictive permissions.
func (g *Generator) SaveSpec(spec []byte, filePath string) error {
	const specFilePerm = 0o600

	return os.WriteFile(filePath, spec, specFilePerm)
}
