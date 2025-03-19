package dto

type CertInfo struct {
	Cert      string `json:"cert" example:"My Trusted Cert"`
	IsTrusted bool   `json:"isTrusted" example:"true"`
}
