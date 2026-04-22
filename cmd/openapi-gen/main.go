// Command openapi-gen writes the OpenAPI specification to doc/openapi.json
// without starting the full application server. Used by CI to publish the
// spec to SwaggerHub; run locally with `make openapi`.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/device-management-toolkit/console/internal/controller/openapi"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const outputPath = "doc/openapi.json"

// NewGeneratorFunc allows tests to inject a fake OpenAPI generator.
var NewGeneratorFunc = func(u usecase.Usecases, l logger.Interface) interface {
	GenerateSpec() ([]byte, error)
	SaveSpec([]byte, string) error
} {
	return openapi.NewGenerator(u, l)
}

func main() {
	l := logger.New("info")

	if err := generate(l); err != nil {
		l.Error("%s", err)
		os.Exit(1)
	}

	l.Info("OpenAPI specification generated at %s", outputPath)
}

func generate(l logger.Interface) error {
	generator := NewGeneratorFunc(usecase.Usecases{}, l)

	spec, err := generator.GenerateSpec()
	if err != nil {
		return fmt.Errorf("generating openapi spec: %w", err)
	}

	const outputDirPerm = 0o755
	if err := os.MkdirAll(filepath.Dir(outputPath), outputDirPerm); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := generator.SaveSpec(spec, outputPath); err != nil {
		return fmt.Errorf("saving openapi spec: %w", err)
	}

	return nil
}
