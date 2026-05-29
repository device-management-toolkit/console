package dto

// PackageAuth selects how rpc-go authenticates to the server.
type PackageAuth struct {
	Mode     string `json:"mode" binding:"required,oneof=token userpass"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// PackageRequest is the body posted to POST /api/package.
type PackageRequest struct {
	Command string      `json:"command" binding:"required,oneof=activate deactivate"`
	Version string      `json:"version" binding:"required"`
	OS      string      `json:"os" binding:"required"`
	Arch    string      `json:"arch" binding:"required"`
	Auth    PackageAuth `json:"auth" binding:"required"`
	Profile string      `json:"profile"`
	Domain  string      `json:"domain"`
}

// RpcAsset is one downloadable build for a release.
type RpcAsset struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// RpcRelease is a single rpc-go release returned to the UI.
type RpcRelease struct {
	Version string     `json:"version"`
	Assets  []RpcAsset `json:"assets"`
}
