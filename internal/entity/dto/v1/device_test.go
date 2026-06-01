package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeviceInfoJSONRoundTrip(t *testing.T) {
	t.Parallel()

	amtEnabled := true
	dhcpEnabled := true
	lmsInstalled := true
	discovered := true
	ethernetAdapterCount := 2
	monitorConnected := true
	ieee8021xEnabled := false
	lastUpdated := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)

	info := DeviceInfo{
		FWVersion:   "16.1.30",
		FWBuild:     "3400",
		FWSku:       "11",
		Discovered:  &discovered,
		CurrentMode: "Admin",
		Features:    "SOL,IDER,KVM",
		IPAddress:   "10.0.0.12",
		LastUpdated: &lastUpdated,
		TLSMode:     "TLS 1.2",
		UPID: map[string]json.RawMessage{
			"oemPlatformIdType": json.RawMessage(`"Not Set (0)"`),
			"oemId":             json.RawMessage(`""`),
			"csmeId":            json.RawMessage(`"4A45A39C5ED94620"`),
		},
		AMTEnabledInBIOS:     &amtEnabled,
		MEInterfaceVersion:   "16.1.25.2124",
		DHCPEnabled:          &dhcpEnabled,
		CertHashes:           []string{"a1b2c3", "d4e5f6"},
		LMSInstalled:         &lmsInstalled,
		LMSVersion:           "2410.5.0.0",
		OSName:               "linux",
		OSVersion:            "6.8.0-51-generic",
		OSDistro:             "Ubuntu 24.04 LTS",
		CPUModel:             "Intel(R) Core(TM) Ultra 7 165H",
		OSIPAddress:          "10.49.76.163",
		EthernetAdapterCount: &ethernetAdapterCount,
		MonitorConnected:     &monitorConnected,
		IEEE8021XEnabled:     &ieee8021xEnabled,
	}

	encoded, err := json.Marshal(info)
	require.NoError(t, err)

	var decoded DeviceInfo
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	require.Equal(t, info.TLSMode, decoded.TLSMode)
	require.Equal(t, info.MEInterfaceVersion, decoded.MEInterfaceVersion)
	require.Equal(t, info.CertHashes, decoded.CertHashes)
	require.Equal(t, info.LMSVersion, decoded.LMSVersion)
	require.NotNil(t, decoded.Discovered)
	require.Equal(t, *info.Discovered, *decoded.Discovered)
	require.NotNil(t, decoded.LMSInstalled)
	require.Equal(t, *info.LMSInstalled, *decoded.LMSInstalled)
}

// TestDevice_JSONContract locks two intentional serialization decisions for the
// identity/lifecycle columns on the /api/v1 device shape:
//   - isDeleted has NO omitempty, so it is always emitted and callers can
//     distinguish a false value from an absent field.
//   - the other new fields ARE omitempty, so they stay absent on empty/legacy
//     payloads and don't change the wire shape existing v1 consumers expect.
func TestDevice_JSONContract(t *testing.T) {
	t.Parallel()

	t.Run("zero value emits isDeleted but omits optional identity fields", func(t *testing.T) {
		t.Parallel()

		out := deviceJSONFields(t, Device{})

		require.Contains(t, out, "isDeleted", "isDeleted must always be present")
		require.JSONEq(t, `false`, string(out["isDeleted"]))

		for _, k := range []string{"id", "createdDate", "deletedDate", "productType", "connectionType"} {
			require.NotContains(t, out, k, "%s must be omitempty on an empty device", k)
		}
	})

	t.Run("populated values serialize under the expected keys", func(t *testing.T) {
		t.Parallel()

		out := deviceJSONFields(t, Device{
			ID:             "id-1",
			CreatedDate:    "2026-05-26T12:00:00Z",
			IsDeleted:      true,
			DeletedDate:    "2026-05-27T08:00:00Z",
			ProductType:    "vpro",
			ConnectionType: "CIRA",
		})

		require.JSONEq(t, `"id-1"`, string(out["id"]))
		require.JSONEq(t, `"2026-05-26T12:00:00Z"`, string(out["createdDate"]))
		require.JSONEq(t, `true`, string(out["isDeleted"]))
		require.JSONEq(t, `"2026-05-27T08:00:00Z"`, string(out["deletedDate"]))
		require.JSONEq(t, `"vpro"`, string(out["productType"]))
		require.JSONEq(t, `"CIRA"`, string(out["connectionType"]))
	})
}

// deviceJSONFields marshals d and returns its top-level JSON object keyed by
// field name, so a test can assert on key presence/absence and values.
func deviceJSONFields(t *testing.T, d Device) map[string]json.RawMessage {
	t.Helper()

	b, err := json.Marshal(d)
	require.NoError(t, err)

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &m))

	return m
}
