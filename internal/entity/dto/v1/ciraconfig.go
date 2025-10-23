package dto

const (
	ServerAddressFormatIPv4 = 3
	ServerAddressFormatIPv6 = 4
	ServerAddressFormatURL  = 201
)

type CIRAConfigCountResponse struct {
	Count int          `json:"totalCount"`
	Data  []CIRAConfig `json:"data"`
}

type CIRAConfig struct {
	ConfigName          string `json:"configName" example:"My CIRA Config"`
	MPSAddress          string `json:"mpsServerAddress" binding:"required,url|ipv4|ipv6" example:"https://example.com"`
	MPSPort             int    `json:"mpsPort" binding:"required,gt=1024,lt=49151" example:"4433"`
	Username            string `json:"username" binding:"required,alphanum" example:"my_username"`
	Password            string `json:"password,omitempty" example:"my_password"`
	CommonName          string `json:"commonName" binding:"required_if=ServerAddressFormat 3" example:"example.com"`
	ServerAddressFormat int    `json:"serverAddressFormat" binding:"required,oneof=3 4 201" example:"201"` // 3 = IPV4, 4= IPV6, 201 = FQDN
	AuthMethod          int    `json:"authMethod" binding:"required,oneof=1 2" example:"2"`                // 1 = Mutal Auth, 2 = Username and Password
	MPSRootCertificate  string `json:"mpsRootCertificate" binding:"required" example:"-----BEGIN CERTIFICATE-----\n..."`
	ProxyDetails        string `json:"proxyDetails" example:"http://example.com"`
	TenantID            string `json:"tenantId" example:"abc123"`
	RegeneratePassword  bool   `json:"regeneratePassword,omitempty" example:"true"`
	Version             string `json:"version,omitempty" example:"1.0.0"`
}
