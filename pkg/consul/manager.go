package consul

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/device-management-toolkit/console/config"
)

func WaitForService(ctx context.Context, svc ServiceManager, serviceName string, logf func(format string, args ...any)) error {
	b := backoff.WithContext(backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(500*time.Millisecond),
		backoff.WithMaxInterval(10*time.Second),
		backoff.WithMaxElapsedTime(0),
	), ctx)

	attempt := 0

	return backoff.Retry(func() error {
		attempt++

		err := svc.Health(ctx, serviceName)
		if err != nil && logf != nil {
			logf("waiting for consul[%d] %v", attempt, err)
		}

		return err
	}, b)
}

func ProcessServiceConfigs(ctx context.Context, svc ServiceManager, cfg *config.Config) error {
	values, err := svc.Get(ctx, cfg.Consul.KeyPrefix)
	if err != nil {
		return fmt.Errorf("consul: get %q: %w", cfg.Consul.KeyPrefix, err)
	}

	if values == nil {
		if err := svc.Seed(ctx, cfg.Consul.KeyPrefix, cfg); err != nil {
			return fmt.Errorf("consul: seed: %w", err)
		}

		return nil
	}

	if err := svc.Process(values, cfg); err != nil {
		return fmt.Errorf("consul: process: %w", err)
	}

	return nil
}
