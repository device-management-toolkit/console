package devices

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SHA-1 is used for thumbprint not signature
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publickey"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publicprivate"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/credential"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

const (
	TypeWireless string = "Wireless"
	TypeTLS      string = "TLS"
	TypeWired    string = "Wired"
)

func processConcreteDependencies(certificateHandle string, profileAssociation *dto.ProfileAssociation, dependencyItems []concrete.ConcreteDependency, securitySettings dto.SecuritySettings) {
	for i := range dependencyItems {
		di := dependencyItems[i]
		if di.Antecedent.ReferenceParameters.SelectorSet.Selectors[0].Text == certificateHandle {
			updateProfileAssociationKey(di.Dependent.ReferenceParameters.SelectorSet.Selectors[0].Text, profileAssociation, securitySettings)
		}
	}
}

// Helper function to update profile association key if matching key is found.
func updateProfileAssociationKey(keyHandle string, profileAssociation *dto.ProfileAssociation, securitySettings dto.SecuritySettings) {
	for _, key := range securitySettings.KeyResponse.Keys {
		if key.InstanceID == keyHandle {
			keyCopy := key
			profileAssociation.Key = &keyCopy

			return
		}
	}
}

func buildCertificateAssociations(profileAssociation dto.ProfileAssociation, securitySettings *dto.SecuritySettings) {
	var publicKeyHandle string

	if profileAssociation.ClientCertificate != nil {
		for i, existingKeyPair := range securitySettings.KeyResponse.Keys {
			if existingKeyPair.InstanceID == profileAssociation.Key.InstanceID {
				securitySettings.KeyResponse.Keys[i].CertificateHandle = profileAssociation.ClientCertificate.InstanceID
				publicKeyHandle = securitySettings.KeyResponse.Keys[i].InstanceID

				break
			}
		}
	}

	certs := securitySettings.CertificateResponse.Certificates
	for i := range certs {
		if (profileAssociation.ClientCertificate == nil || certs[i].InstanceID != profileAssociation.ClientCertificate.InstanceID) &&
			(profileAssociation.RootCertificate == nil || certs[i].InstanceID != profileAssociation.RootCertificate.InstanceID) {
			continue
		}

		if !certs[i].TrustedRootCertificate {
			securitySettings.CertificateResponse.Certificates[i].PublicKeyHandle = publicKeyHandle
		}

		profileAssociationText := getProfileAssociationText(profileAssociation)
		securitySettings.CertificateResponse.Certificates[i].AssociatedProfiles = append(securitySettings.CertificateResponse.Certificates[i].AssociatedProfiles, profileAssociationText)

		break
	}
}

func getProfileAssociationText(profileAssociation dto.ProfileAssociation) string {
	value := profileAssociation.Type
	if profileAssociation.Type == TypeWireless {
		value += " - " + profileAssociation.ProfileID
	}

	return value
}

func buildProfileAssociations(certificateHandle string, profileAssociation *dto.ProfileAssociation, response wsman.Certificates, securitySettings *dto.SecuritySettings) {
	isNewProfileAssociation := true

	for i := range securitySettings.CertificateResponse.Certificates {
		if securitySettings.CertificateResponse.Certificates[i].InstanceID == certificateHandle {
			if securitySettings.CertificateResponse.Certificates[i].TrustedRootCertificate {
				profileAssociation.RootCertificate = &securitySettings.CertificateResponse.Certificates[i]
			} else {
				profileAssociation.ClientCertificate = &securitySettings.CertificateResponse.Certificates[i]
				processConcreteDependencies(certificateHandle, profileAssociation, response.ConcreteDependencyResponse.Items, *securitySettings)
			}
		}
	}

	// Update certificates in the profile association if it already exists in the list
	for i, assoc := range securitySettings.ProfileAssociation {
		if assoc.ProfileID == profileAssociation.ProfileID {
			updateCertificate(&securitySettings.ProfileAssociation[i], profileAssociation)

			isNewProfileAssociation = false

			break
		}
	}

	// If the profile is not in the list, add it
	if isNewProfileAssociation {
		securitySettings.ProfileAssociation = append(securitySettings.ProfileAssociation, *profileAssociation)
	}
}

// Helper function to update certificates if they are not nil.
func updateCertificate(existingAssoc, newAssoc *dto.ProfileAssociation) {
	if newAssoc.RootCertificate != nil {
		existingAssoc.RootCertificate = newAssoc.RootCertificate
	}

	if newAssoc.ClientCertificate != nil {
		existingAssoc.ClientCertificate = newAssoc.ClientCertificate
	}

	if newAssoc.Key != nil {
		existingAssoc.Key = newAssoc.Key
	}
}

func processCertificates(contextItems []credential.CredentialContext, response wsman.Certificates, profileType string, securitySettings *dto.SecuritySettings) {
	for idx := range contextItems {
		var profileAssociation dto.ProfileAssociation

		profileAssociation.Type = profileType
		profileAssociation.ProfileID = strings.TrimPrefix(contextItems[idx].ElementProvidingContext.ReferenceParameters.SelectorSet.Selectors[0].Text, "Intel(r) AMT:IEEE 802.1x Settings ")
		certificateHandle := contextItems[idx].ElementInContext.ReferenceParameters.SelectorSet.Selectors[0].Text

		buildProfileAssociations(certificateHandle, &profileAssociation, response, securitySettings)
		buildCertificateAssociations(profileAssociation, securitySettings)
	}
}

func (uc *UseCase) GetCertificates(c context.Context, guid string) (dto.SecuritySettings, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.SecuritySettings{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.SecuritySettings{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	response, err := device.GetCertificates()
	if err != nil {
		return dto.SecuritySettings{}, err
	}

	securitySettings := dto.SecuritySettings{
		CertificateResponse: CertificatesToDTO(&response.PublicKeyCertificateResponse),
		KeyResponse:         KeysToDTO(&response.PublicPrivateKeyPairResponse),
	}

	if !reflect.DeepEqual(response.CIMCredentialContextResponse, credential.PullResponse{}) {
		processCertificates(response.CIMCredentialContextResponse.Items.CredentialContextTLS, response, TypeTLS, &securitySettings)
		processCertificates(response.CIMCredentialContextResponse.Items.CredentialContext, response, TypeWireless, &securitySettings)
		processCertificates(response.CIMCredentialContextResponse.Items.CredentialContext8021x, response, TypeWired, &securitySettings)
	}

	return securitySettings, nil
}

func CertificatesToDTO(r *publickey.RefinedPullResponse) dto.CertificatePullResponse {
	regex := regexp.MustCompile(`CN=[^,]+`)

	keyManagementItems := make([]dto.RefinedKeyManagementResponse, len(r.KeyManagementItems))
	for i := range r.KeyManagementItems {
		keyManagementItems[i] = dto.RefinedKeyManagementResponse{
			CreationClassName:       keyManagementItems[i].CreationClassName,
			ElementName:             keyManagementItems[i].ElementName,
			EnabledDefault:          keyManagementItems[i].EnabledDefault,
			EnabledState:            keyManagementItems[i].EnabledState,
			Name:                    keyManagementItems[i].Name,
			RequestedState:          keyManagementItems[i].RequestedState,
			SystemCreationClassName: keyManagementItems[i].SystemCreationClassName,
			SystemName:              keyManagementItems[i].SystemName,
		}
	}

	certItems := make([]dto.RefinedCertificate, len(r.PublicKeyCertificateItems))

	for i := range r.PublicKeyCertificateItems {
		displayName := regex.FindString(r.PublicKeyCertificateItems[i].Subject)
		if displayName != "" && len(displayName) >= 2 {
			displayName = displayName[3:]
		} else {
			displayName = r.PublicKeyCertificateItems[i].InstanceID
		}

		certItems[i] = dto.RefinedCertificate{
			ElementName:            r.PublicKeyCertificateItems[i].ElementName,
			InstanceID:             r.PublicKeyCertificateItems[i].InstanceID,
			X509Certificate:        r.PublicKeyCertificateItems[i].X509Certificate,
			TrustedRootCertificate: r.PublicKeyCertificateItems[i].TrustedRootCertificate,
			Issuer:                 r.PublicKeyCertificateItems[i].Issuer,
			Subject:                r.PublicKeyCertificateItems[i].Subject,
			ReadOnlyCertificate:    r.PublicKeyCertificateItems[i].ReadOnlyCertificate,
			PublicKeyHandle:        r.PublicKeyCertificateItems[i].PublicKeyHandle,
			AssociatedProfiles:     r.PublicKeyCertificateItems[i].AssociatedProfiles,
			DisplayName:            displayName,
		}
	}

	return dto.CertificatePullResponse{
		KeyManagementItems: keyManagementItems,
		Certificates:       certItems,
	}
}

func KeysToDTO(r *publicprivate.RefinedPullResponse) dto.KeyPullResponse {
	keyItems := make([]dto.Key, len(r.PublicPrivateKeyPairItems))
	for i, item := range r.PublicPrivateKeyPairItems {
		keyItems[i] = dto.Key{
			ElementName:       item.ElementName,
			InstanceID:        item.InstanceID,
			DERKey:            item.DERKey,
			CertificateHandle: item.CertificateHandle,
		}
	}

	return dto.KeyPullResponse{
		Keys: keyItems,
	}
}

func (uc *UseCase) GetDeviceCertificate(c context.Context, guid string) (dto.Certificate, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.Certificate{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.Certificate{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	cert1, err := device.GetDeviceCertificate()
	if err != nil {
		return dto.Certificate{}, err
	}

	var certDTOs []dto.Certificate

	for _, certBytes := range cert1.Certificate {
		// Parse each certificate byte slice into an x509.Certificate
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			uc.log.Warn(fmt.Sprintf("Failed to parse certificate: %v", err))

			continue
		}

		// Populate the DTO with certificate information
		certDTO := populateCertificateDTO(cert)
		certDTOs = append(certDTOs, certDTO)
	}

	return certDTOs[0], nil
}

func populateCertificateDTO(cert *x509.Certificate) dto.Certificate {
	// Compute the SHA-1 and SHA-256 fingerprints
	sha1Fingerprint := sha1.Sum(cert.Raw) //nolint:gosec // SHA-1 is used for thumbprint not signature
	sha256Fingerprint := sha256.Sum256(cert.Raw)

	// Determine the public key size
	var publicKeySize int

	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		publicKeySize = pub.N.BitLen()
	case *ecdsa.PublicKey:
		publicKeySize = pub.Curve.Params().BitSize
	default:
		publicKeySize = 0 // Unknown or unsupported key type
	}

	// Populate the dto.Certificate struct
	return dto.Certificate{
		CommonName:         cert.Subject.CommonName,
		IssuerName:         cert.Issuer.CommonName,
		SerialNumber:       cert.SerialNumber.String(),
		NotBefore:          cert.NotBefore,
		NotAfter:           cert.NotAfter,
		DNSNames:           cert.DNSNames,
		SHA1Fingerprint:    hex.EncodeToString(sha1Fingerprint[:]),
		SHA256Fingerprint:  hex.EncodeToString(sha256Fingerprint[:]),
		PublicKeyAlgorithm: cert.PublicKeyAlgorithm.String(),
		PublicKeySize:      publicKeySize,
	}
}

func (uc *UseCase) AddCertificate(c context.Context, guid string, certInfo dto.CertInfo) (handle string, err error) {
	var certData []byte

	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return "", err
	}

	if item == nil || item.GUID == "" {
		return "", ErrNotFound
	}

	// Decode base64 certificate
	certData, err = base64.StdEncoding.DecodeString(certInfo.Cert)
	if err != nil {
		return "", err
	}

	// Try to decode as PEM
	block, _ := pem.Decode(certData)
	if block != nil {
		if block.Type != "CERTIFICATE" {
			return "", err
		}

		certData = block.Bytes
	}

	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return "", err
	}

	if cert.NotAfter.Before(time.Now()) {
		return "", err
	}

	pemCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	block, _ = pem.Decode(pemCert)
	if block == nil {
		return "", err
	}

	cleanedCert := strings.ReplaceAll(base64.StdEncoding.EncodeToString(block.Bytes), "\r\n", "")

	device := uc.device.SetupWsmanClient(*item, false, true)

	if certInfo.IsTrusted {
		handle, err = device.AddTrustedRootCert(cleanedCert)
		if err != nil {
			return "", err
		}
	} else {
		handle, err = device.AddClientCert(cleanedCert)
		if err != nil {
			return "", err
		}
	}

	return handle, nil
}
