package consul

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/consul/api"

	"github.com/device-management-toolkit/console/config"
)

const configKeySuffix = "/config"

type Service struct {
	client *api.Client
}

func NewService(host, port string) (*Service, error) {
	if host == "" || port == "" {
		return nil, fmt.Errorf("consul: host and port are required")
	}

	cfg := api.DefaultConfig()
	cfg.Address = fmt.Sprintf("%s:%s", host, port)

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("consul: new client: %w", err)
	}

	return &Service{client: client}, nil
}

func (s *Service) Health(ctx context.Context, serviceName string) error {
	opts := (&api.QueryOptions{}).WithContext(ctx)

	entries, _, err := s.client.Health().Service(serviceName, "", true, opts)
	if err != nil {
		return fmt.Errorf("consul: health: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("consul: service %q has no passing instances", serviceName)
	}

	return nil
}

// Returns (nil, nil) when the prefix has no keys; callers use this to
// decide whether to seed.
func (s *Service) Get(ctx context.Context, prefix string) (map[string][]byte, error) {
	opts := (&api.QueryOptions{}).WithContext(ctx)

	pairs, _, err := s.client.KV().List(prefix, opts)
	if err != nil {
		return nil, fmt.Errorf("consul: kv list %q: %w", prefix, err)
	}

	if len(pairs) == 0 {
		return nil, nil
	}

	out := make(map[string][]byte, len(pairs))
	for _, p := range pairs {
		out[p.Key] = p.Value
	}

	return out, nil
}

func (s *Service) Seed(ctx context.Context, prefix string, cfg *config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("consul: marshal config: %w", err)
	}

	opts := (&api.WriteOptions{}).WithContext(ctx)

	_, err = s.client.KV().Put(&api.KVPair{
		Key:   prefix + configKeySuffix,
		Value: data,
	}, opts)
	if err != nil {
		return fmt.Errorf("consul: kv put: %w", err)
	}

	return nil
}

func (s *Service) Process(values map[string][]byte, cfg *config.Config) error {
	for _, raw := range values {
		if len(raw) == 0 {
			continue
		}

		if err := json.Unmarshal(raw, cfg); err != nil {
			return fmt.Errorf("consul: unmarshal config: %w", err)
		}
	}

	return nil
}
