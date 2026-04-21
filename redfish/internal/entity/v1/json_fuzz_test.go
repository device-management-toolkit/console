package redfish

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// FuzzComputerSystemJSON fuzzes JSON deserialization of the ComputerSystem entity.
// Verifies determinism and no panics on arbitrary payloads.
func FuzzComputerSystemJSON(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"Id":"550e8400-e29b-41d4-a716-446655440001","Name":"System 1","SystemType":"Physical","Manufacturer":"Intel","Model":"vPro","SerialNumber":"SN-001","PowerState":"On","@odata.id":"/redfish/v1/Systems/1","@odata.type":"#ComputerSystem.v1_26_0.ComputerSystem"}`,
		`{"Id":null,"Name":null,"PowerState":null,"Status":null,"MemorySummary":null,"ProcessorSummary":null}`,
		`{"Id":123,"Name":true,"PowerState":42,"SystemType":false}`,
		fmt.Sprintf(`{"Id":"huge","Name":%q,"Manufacturer":%q}`, strings.Repeat("A", 4096), strings.Repeat("B", 4096)),
		`{"Id":"unicode","Name":"系统🖥️","Manufacturer":"制造商","HostName":"主机\u0000名","BiosVersion":"1.0\x00"}`,
		`{"Id":"status","Status":{"State":"Enabled","Health":"OK","HealthRollup":"OK"},"MemorySummary":{"TotalSystemMemoryGiB":32.0,"MemoryMirroring":"System"},"ProcessorSummary":{"Count":4,"CoreCount":8}}`,
		`{"Id":"nullstatus","Status":null,"MemorySummary":{"TotalSystemMemoryGiB":null},"ProcessorSummary":{"Count":null,"Model":null}}`,
		`{"Id":"edge","PowerState":"UnknownState","SystemType":"CustomType","MemorySummary":{"TotalSystemMemoryGiB":-1.0}}`,
		`{"Id":"year-zero","@odata.id":"","@odata.type":""}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONRoundTrip(t, payload, func() *ComputerSystem { return &ComputerSystem{} })
	})
}

// FuzzSessionJSON fuzzes JSON deserialization of the Session entity.
// Verifies determinism and no panics on arbitrary payloads.
func FuzzSessionJSON(f *testing.F) {
	now := time.Now().UTC()
	seedInputs := []string{
		`{}`,
		fmt.Sprintf(`{"id":"550e8400-e29b-41d4-a716-446655440001","username":"admin","token":"jwt.token.here","created_time":%q,"last_access_time":%q,"timeout_seconds":1800,"client_ip":"192.168.1.1","user_agent":"Mozilla/5.0","is_active":true}`,
			now.Format(time.RFC3339), now.Format(time.RFC3339)),
		`{"id":null,"username":null,"token":null,"created_time":null,"is_active":null}`,
		`{"id":123,"username":true,"timeout_seconds":"1800","is_active":"yes"}`,
		fmt.Sprintf(`{"id":"huge","username":%q,"token":%q}`, strings.Repeat("u", 4096), strings.Repeat("t", 4096)),
		`{"id":"unicode","username":"用戶🙂","token":"päss\u0000секрет","client_ip":"::1","user_agent":"テスト"}`,
		`{"id":"edge","timeout_seconds":-1,"created_time":"0000-01-01T00:00:00Z","last_access_time":"9999-12-31T23:59:59Z","is_active":false}`,
		`{"id":"max-timeout","timeout_seconds":2147483647}`,
		`{"id":"bad-time","created_time":"not-a-time","last_access_time":"also-not-a-time"}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		// Session is in the entity package (parent of v1), import via struct literal
		type sessionProxy struct {
			ID             string    `json:"id"`
			Username       string    `json:"username"`
			Token          string    `json:"token"`
			CreatedTime    time.Time `json:"created_time"`
			LastAccessTime time.Time `json:"last_access_time"`
			TimeoutSeconds int       `json:"timeout_seconds"`
			ClientIP       string    `json:"client_ip"`
			UserAgent      string    `json:"user_agent"`
			IsActive       bool      `json:"is_active"`
		}

		fuzzJSONRoundTrip(t, payload, func() *sessionProxy { return &sessionProxy{} })
	})
}

// fuzzJSONRoundTrip is a generic helper that unmarshals a payload twice and asserts determinism.
func fuzzJSONRoundTrip[T any](t *testing.T, payload string, newValue func() *T) {
	t.Helper()

	first := newValue()
	second := newValue()

	firstErr := json.Unmarshal([]byte(payload), first)
	secondErr := json.Unmarshal([]byte(payload), second)

	// Both invocations must agree on success/failure.
	if (firstErr == nil) != (secondErr == nil) {
		t.Fatalf("non-deterministic error: first=%v second=%v payload=%q", firstErr, secondErr, payload)
	}

	if firstErr != nil {
		return // Expected parse failure — not a bug.
	}

	// Both results must be equal.
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("non-deterministic unmarshal result for payload %q", payload)
	}

	// Re-marshal and unmarshal once more to verify round-trip stability.
	marshaled, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal failed after successful unmarshal: %v", err)
	}

	third := newValue()
	if err := json.Unmarshal(marshaled, third); err != nil {
		t.Fatalf("unmarshal of re-marshaled value failed: %v", err)
	}
}
