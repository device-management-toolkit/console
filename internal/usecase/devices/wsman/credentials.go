package wsman

import (
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	vaultDevicePath     = "devices/"
	vaultAMTPasswordKey = "AMT_PASSWORD"
	vaultMPSPasswordKey = "MPS_PASSWORD"

	// amtUsername is the fixed AMT account MPS uses with the Vault password.
	amtUsername = "admin"
)

// objectStorager adds the object read the Vault-backed store provides.
type objectStorager interface {
	security.Storager
	GetObject(key string) (map[string]string, error)
}

// CredentialResolver reads the device secrets RPS writes to Vault at
// "devices/{guid}" instead of sending them over the API. Nil-safe.
type CredentialResolver struct {
	store security.Storager
	log   logger.Interface
}

// NewCredentialResolver returns nil when there is no store to read from.
func NewCredentialResolver(store security.Storager, log logger.Interface) *CredentialResolver {
	if store == nil {
		return nil
	}

	return &CredentialResolver{store: store, log: log}
}

// secret fetches a single field from the device's Vault object, returning "" on any failure.
func (r *CredentialResolver) secret(guid, key string) string {
	if r == nil {
		return ""
	}

	objStore, ok := r.store.(objectStorager)
	if !ok {
		return ""
	}

	secretData, err := objStore.GetObject(vaultDevicePath + guid)
	if err != nil {
		r.log.Warn("Failed to read %s from Vault for device %s: %v", key, guid, err)

		return ""
	}

	return secretData[key]
}

// MPSPassword returns the CIRA password RPS wrote for the device, or "".
func (r *CredentialResolver) MPSPassword(guid string) string {
	return r.secret(guid, vaultMPSPasswordKey)
}

// AMTCredentials mirrors MPS: the fixed AMT username with the Vault password.
func (r *CredentialResolver) AMTCredentials(guid string) (username, password string, ok bool) {
	pwd := r.secret(guid, vaultAMTPasswordKey)
	if pwd == "" {
		return "", "", false
	}

	return amtUsername, pwd, true
}

// ApplyAMT fills in AMT credentials from Vault when the device row has none.
func (r *CredentialResolver) ApplyAMT(device entity.Device) entity.Device {
	if device.Password != "" {
		return device
	}

	if username, password, ok := r.AMTCredentials(device.GUID); ok {
		device.Username = username
		device.Password = password
	}

	return device
}

// DecryptStoredPassword decrypts a password column. An empty column is not
// ciphertext, so it is returned as-is for the Vault fallback.
func DecryptStoredPassword(cryptor security.Cryptor, stored string) (string, error) {
	if stored == "" {
		return "", nil
	}

	return cryptor.Decrypt(stored)
}
