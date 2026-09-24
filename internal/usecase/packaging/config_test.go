package packaging

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func unmarshalConfig(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()

	var m map[string]interface{}

	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("result is not valid YAML: %v\n---\n%s", err, data)
	}

	return m
}

func activateSection(t *testing.T, m map[string]interface{}) map[interface{}]interface{} {
	t.Helper()

	activate, ok := m["activate"].(map[interface{}]interface{})
	if !ok {
		t.Fatalf("activate section missing or wrong type: %T", m["activate"])
	}

	return activate
}

func deactivateSection(t *testing.T, m map[string]interface{}) map[interface{}]interface{} {
	t.Helper()

	deactivate, ok := m["deactivate"].(map[interface{}]interface{})
	if !ok {
		t.Fatalf("deactivate section missing or wrong type: %T", m["deactivate"])
	}

	return deactivate
}

func TestRenderConfigTokenActivateDomain(t *testing.T) {
	t.Parallel()

	req := dto.PackageRequest{
		Command: "activate",
		Version: "v3.0.1",
		OS:      "linux",
		Arch:    "x86_64",
		Auth:    dto.PackageAuth{Mode: "token"},
		Profile: "myProfile",
		Domain:  "corp.com",
	}
	in := configInputs{
		AuthEndpoint:    "https://auth.example.com/token",
		DevicesEndpoint: "https://mps.example.com/devices",
		ExportBase:      "https://rps.example.com",
		AuthToken:       "tok",
	}

	data, err := renderConfig(req, in)
	if err != nil {
		t.Fatalf("renderConfig returned error: %v", err)
	}

	m := unmarshalConfig(t, data)

	if got, _ := m["auth-token"].(string); got != "tok" {
		t.Errorf("auth-token = %q, want %q", got, "tok")
	}

	activate := activateSection(t, m)

	activateURL, _ := activate["url"].(string)

	if !strings.Contains(activateURL, "/profiles/export/myProfile") {
		t.Errorf("activate.url = %q, want it to contain /profiles/export/myProfile", activateURL)
	}

	if !strings.Contains(activateURL, "domainName=corp.com") {
		t.Errorf("activate.url = %q, want it to contain domainName=corp.com", activateURL)
	}
}

func TestRenderConfigUserpassActivateNoDomain(t *testing.T) {
	t.Parallel()

	req := dto.PackageRequest{
		Command: "activate",
		Version: "v3.0.1",
		OS:      "linux",
		Arch:    "x86_64",
		Auth:    dto.PackageAuth{Mode: "userpass", Username: "admin", Password: "secret"},
		Profile: "p1",
		Domain:  "",
	}
	in := configInputs{
		AuthEndpoint:    "https://auth.example.com/token",
		DevicesEndpoint: "https://mps.example.com/devices",
		ExportBase:      "https://rps.example.com",
		AuthToken:       "",
	}

	data, err := renderConfig(req, in)
	if err != nil {
		t.Fatalf("renderConfig returned error: %v", err)
	}

	m := unmarshalConfig(t, data)

	if got, _ := m["auth-username"].(string); got != "admin" {
		t.Errorf("auth-username = %q, want %q", got, "admin")
	}

	if got, _ := m["auth-password"].(string); got != "secret" {
		t.Errorf("auth-password = %q, want %q", got, "secret")
	}

	activate := activateSection(t, m)

	activateURL, _ := activate["url"].(string)

	if !strings.Contains(activateURL, "/export/p1") {
		t.Errorf("activate.url = %q, want it to contain /export/p1", activateURL)
	}

	if strings.Contains(activateURL, "domainName") {
		t.Errorf("activate.url = %q, should not contain domainName when domain is empty", activateURL)
	}
}

func TestRenderConfigTokenDeactivate(t *testing.T) {
	t.Parallel()

	req := dto.PackageRequest{
		Command: "deactivate",
		Version: "v3.0.1",
		OS:      "linux",
		Arch:    "x86_64",
		Auth:    dto.PackageAuth{Mode: "token"},
	}
	in := configInputs{
		AuthEndpoint:    "https://auth.example.com/token",
		DevicesEndpoint: "https://mps.example.com/devices",
		ExportBase:      "https://rps.example.com",
		AuthToken:       "tok",
	}

	data, err := renderConfig(req, in)
	if err != nil {
		t.Fatalf("renderConfig returned error: %v", err)
	}

	m := unmarshalConfig(t, data)

	if got, _ := m["auth-token"].(string); got != "tok" {
		t.Errorf("auth-token = %q, want %q", got, "tok")
	}

	if activate, ok := m["activate"].(map[interface{}]interface{}); ok {
		if activateURL, _ := activate["url"].(string); activateURL != "" {
			t.Errorf("activate.url = %q, want empty for deactivate command", activateURL)
		}
	}

	deactivate := deactivateSection(t, m)

	deactivateURL, _ := deactivate["url"].(string)

	if deactivateURL == "" {
		t.Errorf("deactivate.url is empty, want non-empty")
	}
}

// Mode "none" must leave the credential keys out entirely, not write them empty:
// rpc-go applies config.yaml after env, so an empty key would override the
// AUTH_* variables the operator sets on the device.
func TestRenderConfigNoneOmitsCredentialKeys(t *testing.T) {
	t.Parallel()

	req := dto.PackageRequest{
		Command: "activate",
		Auth:    dto.PackageAuth{Mode: "none"},
		Profile: "p1",
	}

	out, err := renderConfig(req, configInputs{
		AuthEndpoint:    "https://console.example/api/v1/authorize",
		DevicesEndpoint: "https://console.example/api/v1/devices",
		ExportBase:      "https://console.example",
		AuthToken:       "must-not-appear",
	})
	if err != nil {
		t.Fatal(err)
	}

	m := unmarshalConfig(t, out)

	for _, key := range []string{"auth-token", "auth-username", "auth-password"} {
		if _, present := m[key]; present {
			t.Errorf("%s present in config for mode none; it must be omitted", key)
		}
	}

	// rpc-go still needs to know where to exchange the credentials it is given.
	if got, _ := m["auth-endpoint"].(string); got != "https://console.example/api/v1/authorize" {
		t.Errorf("auth-endpoint = %q, want it kept for mode none", got)
	}
}

// Token mode must not leak username/password keys, even empty ones.
func TestRenderConfigTokenOmitsUserPassKeys(t *testing.T) {
	t.Parallel()

	req := dto.PackageRequest{
		Command: "deactivate",
		Auth:    dto.PackageAuth{Mode: "token"},
	}

	out, err := renderConfig(req, configInputs{AuthToken: "tok"})
	if err != nil {
		t.Fatal(err)
	}

	m := unmarshalConfig(t, out)

	if got, _ := m["auth-token"].(string); got != "tok" {
		t.Errorf("auth-token = %q, want %q", got, "tok")
	}

	for _, key := range []string{"auth-username", "auth-password"} {
		if _, present := m[key]; present {
			t.Errorf("%s present in config for token mode", key)
		}
	}
}

func TestRenderConfigEscapesProfileName(t *testing.T) {
	t.Parallel()

	req := dto.PackageRequest{
		Command: "activate",
		Auth:    dto.PackageAuth{Mode: "none"},
		Profile: "My Profile",
		Domain:  "corp",
	}

	out, err := renderConfig(req, configInputs{ExportBase: "https://console.example.com"})
	if err != nil {
		t.Fatal(err)
	}

	got := activateSection(t, unmarshalConfig(t, out))["url"]
	if want := "https://console.example.com/api/v1/admin/profiles/export/My%20Profile?domainName=corp"; got != want {
		t.Errorf("activate url = %v, want %v", got, want)
	}
}
