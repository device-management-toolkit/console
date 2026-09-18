package dto

// PackageAuth selects how rpc-go authenticates to the server. Mode "none"
// embeds no credentials; they are supplied on the device instead.
type PackageAuth struct {
	Mode     string `json:"mode" binding:"required,oneof=token userpass none"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// PackageRequest is the body posted to POST /api/package.
type PackageRequest struct {
	Command  string      `json:"command" binding:"required,oneof=activate deactivate"`
	Version  string      `json:"version" binding:"required"`
	OS       string      `json:"os" binding:"required"` // "windows", "linux", or "both"
	Arch     string      `json:"arch" binding:"required"`
	Auth     PackageAuth `json:"auth" binding:"required"`
	Profile  string      `json:"profile" binding:"required_if=Command activate"`
	Domain   string      `json:"domain"`
	TokenTTL string      `json:"tokenTtl" binding:"omitempty,oneof=15m 1h 8h 24h"`
	// ServerURL is the base URL rpc-go is pointed at, e.g.
	// "https://console.example.com:8181". Empty falls back to the listen address.
	ServerURL string `json:"serverUrl" binding:"omitempty,url"`
}

// RPCAsset is one downloadable build for a release.
type RPCAsset struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// RPCRelease is a single rpc-go release returned to the UI.
type RPCRelease struct {
	Version string     `json:"version"`
	Assets  []RPCAsset `json:"assets"`
}
