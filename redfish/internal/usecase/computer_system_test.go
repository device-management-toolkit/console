package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

type graphicalConsoleTestRepo struct {
	system  *redfishv1.ComputerSystem
	bootErr error
	kvmErr  error
}

func (r *graphicalConsoleTestRepo) GetAll(_ context.Context) ([]string, error) {
	return []string{"system-1"}, nil
}

func (r *graphicalConsoleTestRepo) GetByID(_ context.Context, _ string) (*redfishv1.ComputerSystem, error) {
	if r.system == nil {
		return nil, ErrSystemNotFound
	}

	s := *r.system

	return &s, nil
}

func (r *graphicalConsoleTestRepo) UpdatePowerState(_ context.Context, _ string, _ redfishv1.PowerState) error {
	return nil
}

func (r *graphicalConsoleTestRepo) GetBootSettings(_ context.Context, _ string) (*generated.ComputerSystemBoot, error) {
	if r.bootErr != nil {
		return nil, r.bootErr
	}

	return &generated.ComputerSystemBoot{}, nil
}

func (r *graphicalConsoleTestRepo) UpdateBootSettings(_ context.Context, _ string, _ *generated.ComputerSystemBoot) error {
	return nil
}

func (r *graphicalConsoleTestRepo) UpdateGraphicalConsoleServiceEnabled(_ context.Context, _ string, _ bool) error {
	return r.kvmErr
}

func TestConvertGraphicalConsoleToGeneratedNil(t *testing.T) {
	t.Parallel()

	uc := &ComputerSystemUseCase{}

	if got := uc.convertGraphicalConsoleToGenerated(nil); got != nil {
		t.Fatalf("expected nil GraphicalConsole, got %#v", got)
	}
}

func TestGetComputerSystemGraphicalConsoleMapping(t *testing.T) {
	t.Parallel()

	serviceEnabled := true
	maxSessions := int64(2)
	port := int64(5900)

	repo := &graphicalConsoleTestRepo{
		system: &redfishv1.ComputerSystem{
			ID:         "system-1",
			Name:       "Test System",
			SystemType: redfishv1.SystemTypePhysical,
			PowerState: redfishv1.PowerStateOn,
			GraphicalConsole: &redfishv1.ComputerSystemHostGraphicalConsole{
				ConnectTypesSupported: []string{"KVMIP", "OEM", "INVALID"},
				MaxConcurrentSessions: &maxSessions,
				Port:                  &port,
				ServiceEnabled:        &serviceEnabled,
			},
		},
	}

	uc := &ComputerSystemUseCase{Repo: repo}

	result, err := uc.GetComputerSystem(context.Background(), "system-1")
	if err != nil {
		t.Fatalf("GetComputerSystem returned error: %v", err)
	}

	if result.GraphicalConsole == nil {
		t.Fatal("expected GraphicalConsole to be present")
	}

	if result.GraphicalConsole.Port == nil || *result.GraphicalConsole.Port != 5900 {
		t.Fatalf("expected Port=5900, got %#v", result.GraphicalConsole.Port)
	}

	if result.GraphicalConsole.ServiceEnabled == nil || !*result.GraphicalConsole.ServiceEnabled {
		t.Fatalf("expected ServiceEnabled=true, got %#v", result.GraphicalConsole.ServiceEnabled)
	}

	if result.GraphicalConsole.MaxConcurrentSessions == nil || *result.GraphicalConsole.MaxConcurrentSessions != 2 {
		t.Fatalf("expected MaxConcurrentSessions=2, got %#v", result.GraphicalConsole.MaxConcurrentSessions)
	}

	if result.GraphicalConsole.ConnectTypesSupported == nil {
		t.Fatal("expected ConnectTypesSupported to be set")
	}

	got := *result.GraphicalConsole.ConnectTypesSupported
	if len(got) != 2 {
		t.Fatalf("expected 2 supported connect types, got %d: %#v", len(got), got)
	}

	if got[0] != generated.ComputerSystemGraphicalConnectTypesSupportedKVMIP {
		t.Fatalf("expected first connect type KVMIP, got %q", got[0])
	}

	if got[1] != generated.ComputerSystemGraphicalConnectTypesSupportedOEM {
		t.Fatalf("expected second connect type OEM, got %q", got[1])
	}
}

func TestGetComputerSystemGraphicalConsoleDropsUnsupportedConnectTypes(t *testing.T) {
	t.Parallel()

	serviceEnabled := false

	repo := &graphicalConsoleTestRepo{
		system: &redfishv1.ComputerSystem{
			ID:         "system-1",
			Name:       "Test System",
			SystemType: redfishv1.SystemTypePhysical,
			PowerState: redfishv1.PowerStateOn,
			GraphicalConsole: &redfishv1.ComputerSystemHostGraphicalConsole{
				ConnectTypesSupported: []string{"INVALID"},
				ServiceEnabled:        &serviceEnabled,
			},
		},
		bootErr: errors.New("boot unavailable"),
	}

	uc := &ComputerSystemUseCase{Repo: repo}

	result, err := uc.GetComputerSystem(context.Background(), "system-1")
	if err != nil {
		t.Fatalf("GetComputerSystem returned error: %v", err)
	}

	if result.GraphicalConsole == nil {
		t.Fatal("expected GraphicalConsole to be present")
	}

	if result.GraphicalConsole.ConnectTypesSupported != nil {
		t.Fatalf("expected unsupported connect types to be omitted, got %#v", result.GraphicalConsole.ConnectTypesSupported)
	}
}

func TestUpdateGraphicalConsoleServiceEnabled(t *testing.T) {
	t.Parallel()

	errAMT := errors.New("amt refused")

	tests := []struct {
		name    string
		kvmErr  error
		enabled bool
		wantErr error
	}{
		{
			name:    "success",
			enabled: true,
		},
		{
			name:    "repo error",
			kvmErr:  errAMT,
			enabled: false,
			wantErr: errAMT,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &ComputerSystemUseCase{Repo: &graphicalConsoleTestRepo{kvmErr: tt.kvmErr}}

			err := uc.UpdateGraphicalConsoleServiceEnabled(context.Background(), "system-1", tt.enabled)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateGraphicalConsoleServiceEnabled() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
