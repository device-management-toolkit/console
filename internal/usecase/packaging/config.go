package packaging

import (
	"fmt"

	"gopkg.in/yaml.v2"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// Export path format constants.
const (
	exportPathFmt       = "%s/api/v1/admin/profiles/export/%s"
	exportPathDomainFmt = "%s/api/v1/admin/profiles/export/%s?domainName=%s"
	authModeToken       = "token"
	commandActivate     = "activate"
	commandDeactivate   = "deactivate"
	defaultLMSAddress   = "localhost"
	defaultLMSPort      = "16992"
	defaultLogLevel     = "info"
)

// configFile mirrors the full rpc-go config.yaml structure.
// The configure subtree (amtfeatures/wired/wireless/tls/etc.) is omitted for v1;
// rpc-go ignores unused sections, so this is safe — only populate what you use.
type configFile struct {
	LogLevel        string           `yaml:"log-level"`
	JSON            bool             `yaml:"json"`
	Verbose         bool             `yaml:"verbose"`
	SkipCertCheck   bool             `yaml:"skip-cert-check"`
	LMSAddress      string           `yaml:"lmsaddress"`
	LMSPort         string           `yaml:"lmsport"`
	TenantID        string           `yaml:"tenant-id"`
	AMTPassword     string           `yaml:"amt-password"`
	AuthToken       string           `yaml:"auth-token"`
	AuthUsername    string           `yaml:"auth-username"`
	AuthPassword    string           `yaml:"auth-password"`
	AuthEndpoint    string           `yaml:"auth-endpoint"`
	DevicesEndpoint string           `yaml:"devices-endpoint"`
	AMTInfo         amtInfoConfig    `yaml:"amtinfo"`
	Activate        activateConfig   `yaml:"activate"`
	Deactivate      deactivateConfig `yaml:"deactivate"`
}

// amtInfoConfig maps the amtinfo sub-section of rpc-go config.yaml.
type amtInfoConfig struct {
	Ver              bool   `yaml:"ver"`
	All              bool   `yaml:"all"`
	SKU              bool   `yaml:"sku"`
	UUID             bool   `yaml:"uuid"`
	Mode             bool   `yaml:"mode"`
	DNS              bool   `yaml:"dns"`
	Hostname         bool   `yaml:"hostname"`
	LAN              bool   `yaml:"lan"`
	RAS              bool   `yaml:"ras"`
	OperationalState bool   `yaml:"operationalState"`
	UserCert         bool   `yaml:"userCert"`
	Build            bool   `yaml:"bld"`
	Sync             bool   `yaml:"sync"`
	URL              string `yaml:"url"`
}

// activateConfig maps the activate sub-section of rpc-go config.yaml.
type activateConfig struct {
	Local               bool   `yaml:"local"`
	URL                 string `yaml:"url"`
	Profile             string `yaml:"profile"`
	Proxy               string `yaml:"proxy"`
	CCM                 bool   `yaml:"ccm"`
	ACM                 bool   `yaml:"acm"`
	Key                 string `yaml:"key"`
	DNS                 string `yaml:"dns"`
	Hostname            string `yaml:"hostname"`
	Name                string `yaml:"name"`
	UUID                string `yaml:"uuid"`
	StopConfig          bool   `yaml:"stopConfig"`
	SkipIPRenew         bool   `yaml:"skipIPRenew"`
	ProvisioningCert    string `yaml:"provisioningCert"`
	ProvisioningCertPwd string `yaml:"provisioningCertPwd"`
}

// deactivateConfig maps the deactivate sub-section of rpc-go config.yaml.
type deactivateConfig struct {
	URL     string `yaml:"url"`
	Profile string `yaml:"profile"`
	Proxy   string `yaml:"proxy"`
}

// defaultConfigFile returns a configFile pre-filled with rpc-go sample defaults.
func defaultConfigFile() configFile {
	return configFile{
		LogLevel:   defaultLogLevel,
		LMSAddress: defaultLMSAddress,
		LMSPort:    defaultLMSPort,
	}
}

// configInputs carries caller-resolved values (e.g. pre-minted auth token)
// so that renderConfig remains a pure function.
type configInputs struct {
	AuthEndpoint    string
	DevicesEndpoint string
	ExportBase      string // base URL for the activate profile-export URL
	AuthToken       string // non-empty when auth mode == token
}

// renderConfig builds a complete rpc-go config.yaml from a PackageRequest and
// resolved configInputs. It starts from defaults and applies only the
// request-driven fields; everything else keeps zero/default values.
func renderConfig(req dto.PackageRequest, in configInputs) ([]byte, error) {
	cfg := defaultConfigFile()

	cfg.AuthEndpoint = in.AuthEndpoint
	cfg.DevicesEndpoint = in.DevicesEndpoint

	switch req.Auth.Mode {
	case authModeToken:
		cfg.AuthToken = in.AuthToken
	default:
		cfg.AuthUsername = req.Auth.Username
		cfg.AuthPassword = req.Auth.Password
	}

	switch req.Command {
	case commandActivate:
		if req.Domain != "" {
			cfg.Activate.URL = fmt.Sprintf(exportPathDomainFmt, in.ExportBase, req.Profile, req.Domain)
		} else {
			cfg.Activate.URL = fmt.Sprintf(exportPathFmt, in.ExportBase, req.Profile)
		}
	case commandDeactivate:
		// Remote deactivate targets the server's devices API; the shared auth block carries credentials.
		cfg.Deactivate.URL = in.DevicesEndpoint
	}

	out, err := yaml.Marshal(cfg) //nolint:gosec // G117: config intentionally serializes credential fields (amt-password, auth-token, etc.) — the caller is responsible for protecting the resulting bytes
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	return out, nil
}
