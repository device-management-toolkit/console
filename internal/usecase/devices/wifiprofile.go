package devices

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	cimIEEE8021x "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"

	"github.com/device-management-toolkit/console/internal/repoerrors"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

const (
	defaultWiFiEndpoint             = "WiFi Endpoint 0"
	instanceIDPrefixUserSettings    = "Intel(r) AMT:WiFi Endpoint User Settings"
	instanceIDFormatWiFiEndpoint    = "Intel(r) AMT:WiFi Endpoint Settings %s"
	instanceIDFormatIEEE8021x       = "Intel(r) AMT:IEEE 802.1x Settings %s"
	resourceCIMWiFiEndpointSettings = "CIM_WiFiEndpointSettings"
	resourceCIMIEEE8021xSettings    = "CIM_IEEE8021xSettings"
	selectorNameInstanceID          = "InstanceID"
)

var (
	errInvalidAuthenticationMethod = errors.New("invalid authentication method")
	errInvalidEncryptionMethod     = errors.New("invalid encryption method")
)

type IEEE8021xCertHandles struct {
	ClientCertHandle string
	RootCertHandle   string
}

type preparedWirelessProfile struct {
	wifiRequest      wifi.WiFiEndpointSettingsRequest
	ieee8021xRequest models.IEEE8021xSettings
	certHandles      *IEEE8021xCertHandles
}

func (uc *UseCase) GetWirelessProfiles(c context.Context, guid string) ([]config.WirelessProfile, error) {
	device, err := uc.setupWirelessProfileManagement(c, guid)
	if err != nil {
		return nil, err
	}

	return getWirelessProfilesFromDevice(device)
}

func (uc *UseCase) AddWirelessProfile(c context.Context, guid string, profile config.WirelessProfile) error {
	device, err := uc.setupWirelessProfileManagement(c, guid)
	if err != nil {
		return err
	}

	settings, err := device.GetWiFiSettings()
	if err != nil {
		return err
	}

	if _, found := findWirelessSettingByProfileName(settings, profile.ProfileName); found {
		return wirelessProfileAlreadyExists(profile.ProfileName)
	}

	if _, found := findWirelessSettingByPriority(settings, profile.Priority); found {
		return wirelessProfilePriorityAlreadyExists(profile.Priority)
	}

	preparedProfile, needsPauseBeforeApply, err := prepareWirelessProfileForApply(device, profile)
	if err != nil {
		return err
	}

	if needsPauseBeforeApply {
		if err := waitForAMTCertificateHandling(c, time.Second); err != nil {
			return err
		}
	}

	_, err = device.AddWiFiSettings(
		preparedProfile.wifiRequest,
		preparedProfile.ieee8021xRequest,
		defaultWiFiEndpoint,
		preparedProfile.certHandles.ClientCertHandle,
		preparedProfile.certHandles.RootCertHandle,
	)
	if err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) DeleteWirelessProfile(c context.Context, guid, profileName string) error {
	device, err := uc.setupWirelessProfileManagement(c, guid)
	if err != nil {
		return err
	}

	settings, err := device.GetWiFiSettings()
	if err != nil {
		return err
	}

	setting, found := findWirelessSettingByProfileName(settings, profileName)
	if !found {
		return ErrNotFound
	}

	if err := device.DeleteWiFiSetting(setting.InstanceID); err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) UpdateWirelessProfile(c context.Context, guid string, profile config.WirelessProfile) error {
	device, err := uc.setupWirelessProfileManagement(c, guid)
	if err != nil {
		return err
	}

	settings, err := device.GetWiFiSettings()
	if err != nil {
		return err
	}

	current, found := findWirelessSettingByProfileName(settings, profile.ProfileName)
	if !found {
		return ErrNotFound
	}

	if setting, found := findWirelessSettingByPriority(settings, profile.Priority); found {
		if setting.InstanceID != current.InstanceID {
			return wirelessProfilePriorityAlreadyExists(profile.Priority)
		}
	}

	preparedProfile, needsPauseBeforeApply, err := prepareWirelessProfileForApply(device, profile)
	if err != nil {
		return err
	}

	preparedProfile.wifiRequest.InstanceID = current.InstanceID

	if needsPauseBeforeApply {
		if err := waitForAMTCertificateHandling(c, time.Second); err != nil {
			return err
		}
	}

	_, err = device.UpdateWiFiSettings(
		preparedProfile.wifiRequest,
		preparedProfile.ieee8021xRequest,
		preparedProfile.certHandles.ClientCertHandle,
		preparedProfile.certHandles.RootCertHandle,
	)
	if err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) setupWirelessProfileManagement(c context.Context, guid string) (wsman.Management, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func prepareWirelessProfileForApply(device wsman.Management, profile config.WirelessProfile) (preparedWirelessProfile, bool, error) {
	wifiRequest, err := toWiFiEndpointSettingsRequest(profile)
	if err != nil {
		return preparedWirelessProfile{}, false, err
	}

	prepared := preparedWirelessProfile{
		wifiRequest:      wifiRequest,
		ieee8021xRequest: models.IEEE8021xSettings{},
		certHandles:      &IEEE8021xCertHandles{},
	}

	needsPauseBeforeApply := false

	if profile.IEEE8021x != nil {
		prepared.ieee8021xRequest = toIEEE8021xSettingsRequest(profile)

		certHandles, pauseBeforeAdd, certErr := configureIEEE8021xCertificates(
			device,
			profile.IEEE8021x.PrivateKey,
			profile.IEEE8021x.ClientCert,
			profile.IEEE8021x.CACert,
		)
		if certErr != nil {
			return preparedWirelessProfile{}, false, certErr
		}

		prepared.certHandles = certHandles
		needsPauseBeforeApply = pauseBeforeAdd
	}

	return prepared, needsPauseBeforeApply, nil
}

func wirelessProfileAlreadyExists(profileName string) error {
	notUniqueErr := repoerrors.NotUniqueError{Console: ErrDeviceUseCase}

	return notUniqueErr.Wrap(fmt.Sprintf("wireless profile %q already exists", profileName))
}

func wirelessProfilePriorityAlreadyExists(priority int) error {
	notUniqueErr := repoerrors.NotUniqueError{Console: ErrDeviceUseCase}

	return notUniqueErr.Wrap(fmt.Sprintf("wireless profile with priority %d already exists", priority))
}

func findWirelessSettingByProfileName(settings []wifi.WiFiEndpointSettingsResponse, profileName string) (wifi.WiFiEndpointSettingsResponse, bool) {
	for i := range settings {
		setting := settings[i]
		if setting.InstanceID == "" || isUserSettingsInstanceID(setting.InstanceID) {
			continue
		}

		if setting.ElementName == profileName {
			return setting, true
		}
	}

	return wifi.WiFiEndpointSettingsResponse{}, false
}

func findWirelessSettingByPriority(settings []wifi.WiFiEndpointSettingsResponse, priority int) (wifi.WiFiEndpointSettingsResponse, bool) {
	for i := range settings {
		setting := settings[i]
		if setting.InstanceID == "" || isUserSettingsInstanceID(setting.InstanceID) {
			continue
		}

		if setting.Priority == priority {
			return setting, true
		}
	}

	return wifi.WiFiEndpointSettingsResponse{}, false
}

func getWirelessProfilesFromDevice(device wsman.Management) ([]config.WirelessProfile, error) {
	settings, err := device.GetWiFiSettings()
	if err != nil {
		return nil, err
	}

	ieee8021xResponse, err := device.GetCIMIEEE8021xSettings()
	if err != nil {
		return nil, err
	}

	concreteDependencies, err := device.GetConcreteDependencies()
	if err != nil {
		return nil, err
	}

	ieee8021xByID := indexIEEE8021xSettings(ieee8021xResponse.Body.PullResponse.IEEE8021xSettingsItems)
	ieee8021xByProfileName := indexIEEE8021xSettingsByProfileName(ieee8021xResponse.Body.PullResponse.IEEE8021xSettingsItems)
	associatedIEEE8021xByWiFiID := mapAssociatedIEEE8021xByWiFiID(concreteDependencies)

	profiles := make([]config.WirelessProfile, 0, len(settings))
	for i := range settings {
		setting := settings[i]
		if setting.InstanceID == "" || isUserSettingsInstanceID(setting.InstanceID) {
			continue
		}

		profile := wifiSettingToConfig(setting)
		if ieee8021xSettings, found := findAssociatedIEEE8021xSettings(setting, associatedIEEE8021xByWiFiID, ieee8021xByID, ieee8021xByProfileName); found {
			profile.IEEE8021x = ieee8021xSettingToConfig(ieee8021xSettings)
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func indexIEEE8021xSettings(settings []cimIEEE8021x.IEEE8021xSettingsResponse) map[string]cimIEEE8021x.IEEE8021xSettingsResponse {
	indexed := make(map[string]cimIEEE8021x.IEEE8021xSettingsResponse, len(settings))
	for i := range settings {
		setting := settings[i]
		if setting.InstanceID == "" {
			continue
		}

		indexed[setting.InstanceID] = setting
	}

	return indexed
}

func indexIEEE8021xSettingsByProfileName(settings []cimIEEE8021x.IEEE8021xSettingsResponse) map[string]cimIEEE8021x.IEEE8021xSettingsResponse {
	indexed := make(map[string]cimIEEE8021x.IEEE8021xSettingsResponse, len(settings))
	for i := range settings {
		setting := settings[i]
		if setting.ElementName == "" {
			continue
		}

		indexed[normalizeAssociationKey(setting.ElementName)] = setting
	}

	return indexed
}

func mapAssociatedIEEE8021xByWiFiID(dependencies []concrete.ConcreteDependency) map[string]string {
	associated := map[string]string{}

	for i := range dependencies {
		dependency := dependencies[i]

		wifiEndpointReference, ieee8021xReference, found := dependencyReferencesForWiFi8021x(dependency)
		if !found {
			continue
		}

		wifiID, hasWiFiID := associationReferenceInstanceID(wifiEndpointReference)

		ieee8021xID, hasIEEE8021xID := associationReferenceInstanceID(ieee8021xReference)
		if !hasWiFiID || !hasIEEE8021xID {
			continue
		}

		associated[wifiID] = ieee8021xID
	}

	return associated
}

func dependencyReferencesForWiFi8021x(dependency concrete.ConcreteDependency) (wifiEndpointReference, ieee8021xReference models.AssociationReference, found bool) {
	antecedentURI := dependency.Antecedent.ReferenceParameters.ResourceURI
	dependentURI := dependency.Dependent.ReferenceParameters.ResourceURI

	if isAssociationResource(antecedentURI, resourceCIMWiFiEndpointSettings) && isAssociationResource(dependentURI, resourceCIMIEEE8021xSettings) {
		return dependency.Antecedent, dependency.Dependent, true
	}

	if isAssociationResource(antecedentURI, resourceCIMIEEE8021xSettings) && isAssociationResource(dependentURI, resourceCIMWiFiEndpointSettings) {
		return dependency.Dependent, dependency.Antecedent, true
	}

	return wifiEndpointReference, ieee8021xReference, found
}

func isAssociationResource(resourceURI, resourceName string) bool {
	return strings.HasSuffix(strings.ToLower(resourceURI), strings.ToLower(resourceName))
}

func associationReferenceInstanceID(reference models.AssociationReference) (string, bool) {
	selectors := reference.ReferenceParameters.SelectorSet.Selectors
	for i := range selectors {
		selector := selectors[i]
		if !strings.EqualFold(selector.Name, selectorNameInstanceID) {
			continue
		}

		if selector.Text == "" {
			return "", false
		}

		return selector.Text, true
	}

	return "", false
}

func findAssociatedIEEE8021xSettings(
	setting wifi.WiFiEndpointSettingsResponse,
	associatedIEEE8021xByWiFiID map[string]string,
	ieee8021xByID map[string]cimIEEE8021x.IEEE8021xSettingsResponse,
	ieee8021xByProfileName map[string]cimIEEE8021x.IEEE8021xSettingsResponse,
) (cimIEEE8021x.IEEE8021xSettingsResponse, bool) {
	if ieee8021xID, found := associatedIEEE8021xByWiFiID[setting.InstanceID]; found {
		ieee8021xSettings, exists := ieee8021xByID[ieee8021xID]
		if exists {
			return ieee8021xSettings, true
		}
	}

	if setting.ElementName == "" {
		return cimIEEE8021x.IEEE8021xSettingsResponse{}, false
	}

	fallbackIEEE8021xID := fmt.Sprintf(instanceIDFormatIEEE8021x, setting.ElementName)
	if ieee8021xSettings, found := ieee8021xByID[fallbackIEEE8021xID]; found {
		return ieee8021xSettings, true
	}

	ieee8021xSettings, found := ieee8021xByProfileName[normalizeAssociationKey(setting.ElementName)]

	return ieee8021xSettings, found
}

func normalizeAssociationKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isUserSettingsInstanceID(instanceID string) bool {
	return strings.HasPrefix(instanceID, instanceIDPrefixUserSettings)
}

func waitForAMTCertificateHandling(c context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)

	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	select {
	case <-timer.C:
		return nil
	case <-c.Done():
		return c.Err()
	}
}

func toWiFiEndpointSettingsRequest(req config.WirelessProfile) (wifi.WiFiEndpointSettingsRequest, error) {
	authMethod, ok := wifi.ParseAuthenticationMethod(req.AuthenticationMethod)
	if !ok {
		return wifi.WiFiEndpointSettingsRequest{}, fmt.Errorf("%w %q for profile %q", errInvalidAuthenticationMethod, req.AuthenticationMethod, req.ProfileName)
	}

	encryptionMethod, ok := wifi.ParseEncryptionMethod(req.EncryptionMethod)
	if !ok {
		return wifi.WiFiEndpointSettingsRequest{}, fmt.Errorf("%w %q for profile %q", errInvalidEncryptionMethod, req.EncryptionMethod, req.ProfileName)
	}

	return wifi.WiFiEndpointSettingsRequest{
		ElementName:          req.ProfileName,
		InstanceID:           fmt.Sprintf(instanceIDFormatWiFiEndpoint, req.ProfileName),
		AuthenticationMethod: authMethod,
		EncryptionMethod:     encryptionMethod,
		SSID:                 req.SSID,
		Priority:             req.Priority,
		PSKPassPhrase:        req.Password,
	}, nil
}

func toIEEE8021xSettingsRequest(req config.WirelessProfile) models.IEEE8021xSettings {
	if req.IEEE8021x == nil {
		return models.IEEE8021xSettings{}
	}

	return models.IEEE8021xSettings{
		ElementName:            req.ProfileName,
		InstanceID:             fmt.Sprintf(instanceIDFormatIEEE8021x, req.ProfileName),
		AuthenticationProtocol: models.AuthenticationProtocol(req.IEEE8021x.AuthenticationProtocol),
		Username:               req.IEEE8021x.Username,
		Password:               req.IEEE8021x.Password,
	}
}

func wifiSettingToConfig(setting wifi.WiFiEndpointSettingsResponse) config.WirelessProfile {
	return config.WirelessProfile{
		ProfileName:          setting.ElementName,
		SSID:                 setting.SSID,
		AuthenticationMethod: setting.AuthenticationMethod.String(),
		EncryptionMethod:     setting.EncryptionMethod.String(),
		Priority:             setting.Priority,
	}
}

func ieee8021xSettingToConfig(setting cimIEEE8021x.IEEE8021xSettingsResponse) *config.IEEE8021x {
	return &config.IEEE8021x{
		Username:               setting.Username,
		Password:               setting.Password,
		AuthenticationProtocol: setting.AuthenticationProtocol,
	}
}

func configureIEEE8021xCertificates(
	device wsman.Management,
	privateKey, clientCert, caCert string,
) (*IEEE8021xCertHandles, bool, error) {
	handles := &IEEE8021xCertHandles{}

	certs, err := device.GetCertificates()
	if err != nil {
		return nil, false, err
	}

	addedCredentials := false

	if privateKey != "" {
		var added bool

		_, certs, added, err = resolveOrAddCredentialHandle(certs, privateKey, findExistingPrivateKeyHandle, device.AddPrivateKey, device.GetCertificates)
		if err != nil {
			return nil, false, err
		}

		addedCredentials = addedCredentials || added
	}

	if clientCert != "" {
		var added bool

		handles.ClientCertHandle, certs, added, err = resolveOrAddCredentialHandle(certs, clientCert, findExistingClientCertHandle, device.AddClientCert, device.GetCertificates)
		if err != nil {
			return nil, false, err
		}

		addedCredentials = addedCredentials || added
	}

	if caCert != "" {
		var added bool

		handles.RootCertHandle, _, added, err = resolveOrAddCredentialHandle(certs, caCert, findExistingTrustedRootCertHandle, device.AddTrustedRootCert, device.GetCertificates)
		if err != nil {
			return nil, false, err
		}

		addedCredentials = addedCredentials || added
	}

	return handles, addedCredentials, nil
}

type (
	credentialHandleFinder func(certs wsman.Certificates, credential string) (string, bool)
	credentialHandleAdder  func(credential string) (string, error)
	certsRefresher         func() (wsman.Certificates, error)
)

func resolveOrAddCredentialHandle(certs wsman.Certificates, credential string, find credentialHandleFinder, add credentialHandleAdder, refresh certsRefresher) (handle string, updatedCerts wsman.Certificates, added bool, err error) {
	updatedCerts = certs

	if credential == "" {
		return "", updatedCerts, false, nil
	}

	handle, found := find(updatedCerts, credential)
	if found {
		return handle, updatedCerts, false, nil
	}

	handle, addErr := add(credential)
	if addErr == nil {
		return handle, updatedCerts, true, nil
	}

	if !strings.Contains(strings.ToLower(addErr.Error()), "already exists") {
		return "", updatedCerts, false, addErr
	}

	updatedCerts, err = refresh()
	if err != nil {
		return "", updatedCerts, false, err
	}

	handle, found = find(updatedCerts, credential)
	if !found {
		return "", updatedCerts, false, addErr
	}

	return handle, updatedCerts, false, nil
}

func findExistingPrivateKeyHandle(certs wsman.Certificates, privateKey string) (string, bool) {
	for i := range certs.PublicPrivateKeyPairResponse.PublicPrivateKeyPairItems {
		item := certs.PublicPrivateKeyPairResponse.PublicPrivateKeyPairItems[i]
		if item.DERKey == privateKey {
			return item.InstanceID, true
		}
	}

	return "", false
}

func findExistingClientCertHandle(certs wsman.Certificates, clientCert string) (string, bool) {
	for i := range certs.PublicKeyCertificateResponse.PublicKeyCertificateItems {
		item := certs.PublicKeyCertificateResponse.PublicKeyCertificateItems[i]
		if item.X509Certificate == clientCert && !item.TrustedRootCertificate {
			return item.InstanceID, true
		}
	}

	return "", false
}

func findExistingTrustedRootCertHandle(certs wsman.Certificates, caCert string) (string, bool) {
	for i := range certs.PublicKeyCertificateResponse.PublicKeyCertificateItems {
		item := certs.PublicKeyCertificateResponse.PublicKeyCertificateItems[i]
		if item.X509Certificate == caCert && item.TrustedRootCertificate {
			return item.InstanceID, true
		}
	}

	return "", false
}
