package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeviceJSONOmitsPasswordProperties(t *testing.T) {
	t.Parallel()

	device := Device{
		GUID:         "4c4c4544-0046-3510-8050-c2c04f365033",
		Username:     "admin",
		Password:     "AmtP@ss123",
		MPSPassword:  "P@ssw0rd!",
		MEBXPassword: "mebxsecret",
	}

	for name, encode := range map[string]func() ([]byte, error){
		"value":   func() ([]byte, error) { return json.Marshal(device) },
		"pointer": func() ([]byte, error) { return json.Marshal(&device) },
		"nested":  func() ([]byte, error) { return json.Marshal(DeviceCountResponse{Count: 1, Data: []Device{device}}) },
	} {
		encoded, err := encode()
		require.NoError(t, err, name)
		require.Contains(t, string(encoded), `"username":"admin"`, name)
		require.NotContains(t, string(encoded), "password", name)
		require.NotContains(t, string(encoded), "AmtP@ss123", name)
		require.NotContains(t, string(encoded), "P@ssw0rd!", name)
		require.NotContains(t, string(encoded), "mebxsecret", name)
	}

	require.Equal(t, "AmtP@ss123", device.Password)
}

func TestDeviceJSONDecodesPasswordProperties(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"guid":"4c4c4544-0046-3510-8050-c2c04f365033","password":"AmtP@ss123","mpspassword":"P@ssw0rd!","mebxpassword":"mebxsecret"}`)

	var decoded Device
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, "AmtP@ss123", decoded.Password)
	require.Equal(t, "P@ssw0rd!", decoded.MPSPassword)
	require.Equal(t, "mebxsecret", decoded.MEBXPassword)
}

func TestDeviceInfoJSONRoundTrip(t *testing.T) {
	t.Parallel()

	amtEnabled := true
	dhcpEnabled := true
	lmsInstalled := true
	discovered := true
	ethernetAdapterCount := 2
	monitorConnected := true
	ieee8021xEnabled := false
	firstDiscovered := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	lastSynced := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)

	info := DeviceInfo{
		FWVersion:       "16.1.30",
		FWBuild:         "3400",
		FWSku:           "11",
		Discovered:      &discovered,
		FirstDiscovered: &firstDiscovered,
		CurrentMode:     "Admin",
		Features:        "SOL,IDER,KVM",
		IPAddress:       "10.0.0.12",
		LastSynced:      &lastSynced,
		TLSMode:         "TLS 1.2",
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
	require.NotNil(t, decoded.FirstDiscovered)
	require.Equal(t, *info.FirstDiscovered, *decoded.FirstDiscovered)
	require.NotNil(t, decoded.LastSynced)
	require.Equal(t, *info.LastSynced, *decoded.LastSynced)
}

func TestDeviceInfoJSONLegacyLastUpdatedMapsToLastSynced(t *testing.T) {
	t.Parallel()

	legacy := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	payload := []byte(`{"fwVersion":"16.1.30","lastUpdated":"` + legacy.Format(time.RFC3339Nano) + `"}`)

	var decoded DeviceInfo
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.NotNil(t, decoded.LastSynced)
	require.Equal(t, legacy, *decoded.LastSynced)
}

func TestDeviceInfoJSONLegacyPlatformAdapterNamesDecodeAsArrays(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"platformAdapters":{"wired":"eth0","wireless":"wlan0"}}`)

	var decoded DeviceInfo
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.NotNil(t, decoded.PlatformAdapters)
	require.Equal(t, []string{"eth0"}, decoded.PlatformAdapters.Wired)
	require.Equal(t, []string{"wlan0"}, decoded.PlatformAdapters.Wireless)
}

func TestDeviceInfoJSONPlatformAdapterNameArraysDecode(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"platformAdapters":{"wired":["eth0","eth1"],"wireless":["wlan0","wlan1"]}}`)

	var decoded DeviceInfo
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.NotNil(t, decoded.PlatformAdapters)
	require.Equal(t, []string{"eth0", "eth1"}, decoded.PlatformAdapters.Wired)
	require.Equal(t, []string{"wlan0", "wlan1"}, decoded.PlatformAdapters.Wireless)
}
