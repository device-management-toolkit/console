package packaging

import (
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// The derived fallback URL must match the scheme the listener actually serves,
// or the generated config points rpc-go at the wrong protocol.
func TestBuildConfigInputsDerivedSchemeFollowsTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tlsEnabled    bool
		certFile      string
		wantPrefix    string
		wantSkipCerts bool
	}{
		{"tls off yields http", false, "", "http://localhost:8181", false},
		{"tls on yields https and skips cert check for self-signed", true, "", "https://localhost:8181", true},
		{"configured cert keeps verification on", true, "/etc/console/tls.crt", "https://localhost:8181", false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig("")
			cfg.TLS = config.TLS{Enabled: tt.tlsEnabled, CertFile: tt.certFile}

			svc := New(cfg, logger.New("error"))

			in, err := svc.buildConfigInputs(dto.PackageRequest{
				Command: "activate",
				Auth:    dto.PackageAuth{Mode: "userpass"},
			}, "")
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(in.AuthEndpoint, tt.wantPrefix) {
				t.Errorf("AuthEndpoint = %q, want prefix %q", in.AuthEndpoint, tt.wantPrefix)
			}

			if in.SkipCertCheck != tt.wantSkipCerts {
				t.Errorf("SkipCertCheck = %v, want %v", in.SkipCertCheck, tt.wantSkipCerts)
			}
		})
	}
}

// A server URL from the request still points at this listener, so a
// self-signed cert must keep the cert check off.
func TestBuildConfigInputsServerURLFollowsTLSCert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		certFile      string
		wantSkipCerts bool
	}{
		{"self-signed cert skips cert check", "", true},
		{"configured cert keeps verification on", "/etc/console/tls.crt", false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig("")
			cfg.TLS = config.TLS{Enabled: true, CertFile: tt.certFile}

			svc := New(cfg, logger.New("error"))

			in, err := svc.buildConfigInputs(dto.PackageRequest{
				Command:   "activate",
				Auth:      dto.PackageAuth{Mode: "userpass"},
				ServerURL: "https://console.example.com",
			}, "")
			if err != nil {
				t.Fatal(err)
			}

			if in.AuthEndpoint != "https://console.example.com/api/v1/authorize" {
				t.Errorf("AuthEndpoint = %q", in.AuthEndpoint)
			}

			if in.SkipCertCheck != tt.wantSkipCerts {
				t.Errorf("SkipCertCheck = %v, want %v", in.SkipCertCheck, tt.wantSkipCerts)
			}
		})
	}
}

// The request tenant must reach the generated config, or a package built by one
// tenant resolves its profile against another tenant's data.
func TestRenderConfigCarriesTenant(t *testing.T) {
	t.Parallel()

	svc := New(newTestConfig(""), logger.New("error"))

	req := dto.PackageRequest{
		Command: "activate",
		Auth:    dto.PackageAuth{Mode: "userpass", Username: "u", Password: "p"},
		Profile: "profile1",
	}

	in, err := svc.buildConfigInputs(req, "acme-corp")
	if err != nil {
		t.Fatal(err)
	}

	if in.TenantID != "acme-corp" {
		t.Fatalf("TenantID = %q, want %q", in.TenantID, "acme-corp")
	}

	out, err := renderConfig(req, in)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(out), "tenant-id: acme-corp") {
		t.Errorf("rendered config missing tenant-id:\n%s", out)
	}
}
