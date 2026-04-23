package consul

import (
	"context"

	"github.com/device-management-toolkit/console/config"
)

type ServiceManager interface {
	Health(ctx context.Context, serviceName string) error
	Get(ctx context.Context, prefix string) (map[string][]byte, error)
	Seed(ctx context.Context, prefix string, cfg *config.Config) error
	Process(values map[string][]byte, cfg *config.Config) error
}
