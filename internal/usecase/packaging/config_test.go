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
