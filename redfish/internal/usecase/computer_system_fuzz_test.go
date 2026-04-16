package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

const fuzzTestSystemID = "550e8400-e29b-41d4-a716-446655440001"

// fuzzMockRepo is a minimal in-memory ComputerSystemRepository for fuzz tests.
// It avoids importing the mocks package (which would create an import cycle).
type fuzzMockRepo struct {
	systems map[string]*redfishv1.ComputerSystem
}

func newFuzzMockRepo() *fuzzMockRepo {
	r := &fuzzMockRepo{systems: make(map[string]*redfishv1.ComputerSystem)}
	r.systems[fuzzTestSystemID] = &redfishv1.ComputerSystem{
		ID:         fuzzTestSystemID,
		Name:       "Fuzz Test System",
		SystemType: redfishv1.SystemTypePhysical,
		PowerState: redfishv1.PowerStateOn,
		Status:     &redfishv1.Status{State: "Enabled", Health: "OK"},
	}

	return r
}

func (r *fuzzMockRepo) AddSystem(s *redfishv1.ComputerSystem) { r.systems[s.ID] = s }

func (r *fuzzMockRepo) GetAll(_ context.Context) ([]string, error) {
	ids := make([]string, 0, len(r.systems))
	for id := range r.systems {
		ids = append(ids, id)
	}

	return ids, nil
}

func (r *fuzzMockRepo) GetByID(_ context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	s, ok := r.systems[systemID]
	if !ok {
		return nil, ErrSystemNotFound
	}

	cp := *s

	return &cp, nil
}

func (r *fuzzMockRepo) UpdatePowerState(_ context.Context, systemID string, state redfishv1.PowerState) error {
	s, ok := r.systems[systemID]
	if !ok {
		return ErrSystemNotFound
	}

	s.PowerState = state

	return nil
}

func (r *fuzzMockRepo) GetBootSettings(_ context.Context, _ string) (*generated.ComputerSystemBoot, error) {
	return &generated.ComputerSystemBoot{}, nil
}

func (r *fuzzMockRepo) UpdateBootSettings(_ context.Context, systemID string, _ *generated.ComputerSystemBoot) error {
	if _, ok := r.systems[systemID]; !ok {
		return ErrSystemNotFound
	}

	return nil
}

// newFuzzUseCase returns a ComputerSystemUseCase backed by the inline mock repository.
func newFuzzUseCase() *ComputerSystemUseCase {
	return &ComputerSystemUseCase{Repo: newFuzzMockRepo()}
}

// FuzzComputerSystemTransforms fuzzes GetComputerSystem with arbitrary systemID values.
// Verifies: no panics, deterministic results for the same ID, nil safety in conversions.
func FuzzComputerSystemTransforms(f *testing.F) {
	seeds := []string{
		fuzzTestSystemID,
		"",
		"not-a-uuid",
		"00000000-0000-0000-0000-000000000000",
		strings.Repeat("a", 4096),
		"../etc/passwd",
		"550e8400-e29b-41d4-a716-446655440001\x00extra",
		"용戶-🙂-секрет",
		"UPPERCASE-UUID-550E8400-E29B-41D4-A716-446655440001",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	uc := newFuzzUseCase()
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, systemID string) {
		// Call twice — must be deterministic.
		result1, err1 := uc.GetComputerSystem(ctx, systemID)
		result2, err2 := uc.GetComputerSystem(ctx, systemID)

		// Both must agree on error state.
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("non-deterministic error for systemID %q: first=%v second=%v", systemID, err1, err2)
		}

		if err1 != nil {
			// Errors are expected for invalid IDs.
			return
		}

		// Both results must be non-nil on success.
		if result1 == nil || result2 == nil {
			t.Fatal("expected non-nil result on success")

			return
		}

		// ID field must match the input.
		if result1.Id != systemID {
			t.Fatalf("expected Id %q, got %q", systemID, result1.Id)
		}
	})
}

// FuzzSetPowerState fuzzes SetPowerState with arbitrary reset type strings.
// Verifies: no panics, consistent error handling, valid/invalid reset types handled correctly.
func FuzzSetPowerState(f *testing.F) {
	validResetTypes := []string{
		string(generated.ResourceResetTypeOn),
		string(generated.ResourceResetTypeForceOff),
		string(generated.ResourceResetTypeForceRestart),
		string(generated.ResourceResetTypeGracefulShutdown),
		string(generated.ResourceResetTypeGracefulRestart),
		string(generated.ResourceResetTypePowerCycle),
	}

	invalidResetTypes := []string{
		"",
		"invalid",
		"FORCEON",
		strings.Repeat("X", 4096),
		"On\x00",
		"用戶",
		"<script>alert(1)</script>",
	}

	for _, rt := range append(validResetTypes, invalidResetTypes...) {
		f.Add(rt)
	}

	uc := newFuzzUseCase()
	ctx := context.Background()

	f.Fuzz(func(t *testing.T, resetTypeStr string) {
		resetType := generated.ResourceResetType(resetTypeStr)

		// Must never panic regardless of input.
		err1 := uc.SetPowerState(ctx, fuzzTestSystemID, resetType)
		// Call twice — deterministic error state (ignoring state changes from first call).
		uc2 := newFuzzUseCase()
		err2 := uc2.SetPowerState(ctx, fuzzTestSystemID, resetType)

		// Both should either succeed or both fail (same reset type, fresh repo each time).
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("non-deterministic error for resetType %q: first=%v second=%v", resetTypeStr, err1, err2)
		}
	})
}

// FuzzUpdateBootSettings fuzzes UpdateBootSettings with arbitrary boot configuration inputs.
// Verifies: no panics, deterministic error behavior on invalid boot settings.
func FuzzUpdateBootSettings(f *testing.F) {
	type seed struct {
		bootTarget  string
		bootEnabled string
		bootMode    string
	}

	seeds := []seed{
		{"Pxe", "Once", "UEFI"},
		{"Hdd", "Continuous", "Legacy"},
		{"None", "Disabled", "UEFI"},
		{"", "", ""},
		{"InvalidTarget", "InvalidEnabled", "InvalidMode"},
		{strings.Repeat("X", 4096), strings.Repeat("Y", 4096), strings.Repeat("Z", 4096)},
		{"Pxe\x00", "Once\x00", "UEFI\x00"},
		{"用戶", "секрет", "🙂"},
	}

	for _, s := range seeds {
		f.Add(s.bootTarget, s.bootEnabled, s.bootMode)
	}

	ctx := context.Background()

	f.Fuzz(func(t *testing.T, bootTarget, bootEnabled, bootMode string) {
		uc := newFuzzUseCase()

		target := generated.ComputerSystemBoot_BootSourceOverrideTarget{}
		if err := target.FromComputerSystemBootSource(generated.ComputerSystemBootSource(bootTarget)); err != nil {
			// Invalid target type — expected, not a bug.
			return
		}

		enabled := generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
		if err := enabled.FromComputerSystemBootSourceOverrideEnabled(generated.ComputerSystemBootSourceOverrideEnabled(bootEnabled)); err != nil {
			return
		}

		mode := generated.ComputerSystemBoot_BootSourceOverrideMode{}
		if err := mode.FromComputerSystemBootSourceOverrideMode(generated.ComputerSystemBootSourceOverrideMode(bootMode)); err != nil {
			return
		}

		boot := &generated.ComputerSystemBoot{
			BootSourceOverrideTarget:  &target,
			BootSourceOverrideEnabled: &enabled,
			BootSourceOverrideMode:    &mode,
		}

		err1 := uc.UpdateBootSettings(ctx, fuzzTestSystemID, boot)
		uc2 := newFuzzUseCase()
		err2 := uc2.UpdateBootSettings(ctx, fuzzTestSystemID, boot)

		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("non-deterministic error for boot settings: first=%v second=%v", err1, err2)
		}
	})
}

// FuzzPowerStateConversion fuzzes the PowerState field mapping in GetComputerSystem.
// Injects arbitrary PowerState strings via a custom mock repo and verifies no panics.
func FuzzPowerStateConversion(f *testing.F) {
	powerStates := []string{
		string(redfishv1.PowerStateOn),
		string(redfishv1.PowerStateOff),
		string(redfishv1.ResetTypeForceOff),
		string(redfishv1.ResetTypeForceRestart),
		string(redfishv1.ResetTypePowerCycle),
		"",
		"Unknown",
		"STANDBY",
		strings.Repeat("P", 4096),
		"On\x00",
		"🔌",
	}

	for _, ps := range powerStates {
		f.Add(ps)
	}

	ctx := context.Background()

	f.Fuzz(func(t *testing.T, powerStateStr string) {
		// Build a fresh mock repo with the fuzzed power state injected.
		repo := newFuzzMockRepo()
		system := &redfishv1.ComputerSystem{
			ID:         fuzzTestSystemID,
			Name:       "Fuzz System",
			PowerState: redfishv1.PowerState(powerStateStr),
			SystemType: redfishv1.SystemTypePhysical,
		}
		repo.AddSystem(system)

		uc := &ComputerSystemUseCase{Repo: repo}

		// Must not panic regardless of PowerState value.
		result, err := uc.GetComputerSystem(ctx, fuzzTestSystemID)
		if err != nil {
			return
		}

		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})
}
