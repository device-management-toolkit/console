package dto

type CertInfo struct {
	Cert      string `json:"cert" binding:"required" example:"-----BEGIN CERTIFICATE-----\n..."`
	IsTrusted bool   `json:"isTrusted" example:"true"`
}
