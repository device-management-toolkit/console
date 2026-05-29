package packaging

import (
	"context"
	"io"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// Feature is the public contract for the packaging usecase.
type Feature interface {
	ListVersions(ctx context.Context) ([]dto.RPCRelease, error)
	BuildPackage(ctx context.Context, req dto.PackageRequest) (io.Reader, string, error)
}
