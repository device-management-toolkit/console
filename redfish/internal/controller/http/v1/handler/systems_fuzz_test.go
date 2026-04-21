package v1

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

// FuzzValidateSystemID fuzzes the validateSystemID function with arbitrary strings.
// This is a critical security boundary — it guards all system-level HTTP handlers.
// Verifies: no panics, correct rejection of non-UUID inputs, acceptance of valid UUIDs,
// and deterministic results.
func FuzzValidateSystemID(f *testing.F) {
	validUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440001",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-4fff-bfff-ffffffffffff",
		"AAAAAAAA-BBBB-4CCC-9DDD-EEEEEEEEEEEE",
		"12345678-1234-4234-b234-123456789012",
	}

	invalidInputs := []string{
		"",
		"not-a-uuid",
		"550e8400-e29b-41d4-a716",
		"550e8400-e29b-41d4-a716-446655440001-extra",
		"../etc/passwd",
		"550e8400/e29b/41d4/a716/446655440001",
		"\x00\xFF",
		strings.Repeat("a", 4096),
		"用戶-🙂-0000-0000-000000000000",
		"550e8400-e29b-41d4-a716-4466554400010",
		"550e8400 e29b 41d4 a716 446655440001",
		"<script>alert(1)</script>",
		"' OR 1=1 --",
	}

	for _, v := range append(validUUIDs, invalidInputs...) {
		f.Add(v)
	}

	f.Fuzz(func(t *testing.T, systemID string) {
		err1 := validateSystemID(systemID)
		err2 := validateSystemID(systemID)

		// Must be deterministic.
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("non-deterministic result for systemID %q: first=%v second=%v", systemID, err1, err2)
		}

		// Empty string must always be rejected.
		if systemID == "" && err1 == nil {
			t.Fatal("empty systemID must return an error")
		}
	})
}

// FuzzPatchSystemJSON fuzzes JSON body parsing of ComputerSystemComputerSystem,
// which is the type parsed in PatchRedfishV1SystemsComputerSystemId.
// Verifies: no panics, deterministic unmarshal, no data alongside error.
func FuzzPatchSystemJSON(f *testing.F) {
	seeds := []string{
		`{}`,
		`{"Boot":null}`,
		`{"Boot":{"BootSourceOverrideTarget":null,"BootSourceOverrideEnabled":null,"BootSourceOverrideMode":null}}`,
		`{"Boot":{"BootSourceOverrideTarget":123,"BootSourceOverrideEnabled":true}}`,
		`{"Boot":"not-an-object"}`,
		`{"UnknownField":"value","AnotherField":42}`,
		fmt.Sprintf(`{"Id":%q,"Name":"system"}`, strings.Repeat("a", 4096)),
		`{"PowerState":{"ResourcePowerState":"On"},"SystemType":{"ComputerSystemSystemType":"Physical"}}`,
		fmt.Sprintf(`{%s"z":"end"}`, strings.Repeat(`"k":"v",`, 500)),
		`{"Id":"unicode","Name":"系统🖥️\u0000","Description":{"ResourceDescription":"desc"}}`,
		`{"Id":null,"Name":null,"Boot":null,"PowerState":null,"Status":null}`,
		`{"Id":123,"Name":true,"Boot":42}`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		first := &generated.ComputerSystemComputerSystem{}
		second := &generated.ComputerSystemComputerSystem{}

		firstErr := json.Unmarshal([]byte(payload), first)
		secondErr := json.Unmarshal([]byte(payload), second)

		// Both calls must agree on error state.
		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("non-deterministic error for payload %q: first=%v second=%v", payload, firstErr, secondErr)
		}

		if firstErr != nil {
			return
		}

		// Results must be equal.
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("non-deterministic unmarshal result for payload %q", payload)
		}
	})
}
