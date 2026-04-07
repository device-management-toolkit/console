package devices

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publickey"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publicprivate"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	cimIEEE8021x "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/ieee8021x"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"

	"github.com/device-management-toolkit/console/internal/repoerrors"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

const testNewHandle = "new-handle"

func wifiProfileTestCertificates() wsman.Certificates {
	return wsman.Certificates{
		PublicPrivateKeyPairResponse: publicprivate.RefinedPullResponse{
			PublicPrivateKeyPairItems: []publicprivate.RefinedPublicPrivateKeyPair{{
				InstanceID: "pk-handle",
				DERKey:     "private-key",
			}},
		},
		PublicKeyCertificateResponse: publickey.RefinedPullResponse{
			PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{
				{InstanceID: "client-handle", X509Certificate: "client-cert", TrustedRootCertificate: false},
				{InstanceID: "root-handle", X509Certificate: "ca-cert", TrustedRootCertificate: true},
			},
		},
	}
}

func TestWiFiProfileTransformers(t *testing.T) {
	t.Parallel()

	t.Run("toWiFiEndpointSettingsRequest", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			profile config.WirelessProfile
			res     wifi.WiFiEndpointSettingsRequest
			err     string
		}{
			{
				name: "success",
				profile: config.WirelessProfile{
					ProfileName:          "Office",
					SSID:                 "OfficeSSID",
					Password:             "P@ssword",
					AuthenticationMethod: "WPA2PSK",
					EncryptionMethod:     "CCMP",
					Priority:             5,
				},
				res: wifi.WiFiEndpointSettingsRequest{
					ElementName:          "Office",
					InstanceID:           "Intel(r) AMT:WiFi Endpoint Settings Office",
					AuthenticationMethod: wifi.AuthenticationMethodWPA2PSK,
					EncryptionMethod:     wifi.EncryptionMethodCCMP,
					SSID:                 "OfficeSSID",
					Priority:             5,
					PSKPassPhrase:        "P@ssword",
				},
			},
			{
				name: "invalid authentication method",
				profile: config.WirelessProfile{
					ProfileName:          "Office",
					AuthenticationMethod: "INVALID",
					EncryptionMethod:     "CCMP",
				},
				err: "invalid authentication method \"INVALID\" for profile \"Office\"",
			},
			{
				name: "invalid encryption method",
				profile: config.WirelessProfile{
					ProfileName:          "Office",
					AuthenticationMethod: "WPA2PSK",
					EncryptionMethod:     "INVALID",
				},
				err: "invalid encryption method \"INVALID\" for profile \"Office\"",
			},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				res, err := toWiFiEndpointSettingsRequest(tc.profile)
				if tc.err != "" {
					require.EqualError(t, err, tc.err)
					require.Equal(t, wifi.WiFiEndpointSettingsRequest{}, res)

					return
				}

				require.NoError(t, err)
				require.Equal(t, tc.res, res)
			})
		}
	})

	t.Run("toIEEE8021xSettingsRequest", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			profile config.WirelessProfile
			res     models.IEEE8021xSettings
		}{
			{
				name:    "empty",
				profile: config.WirelessProfile{},
				res:     models.IEEE8021xSettings{},
			},
			{
				name: "success",
				profile: config.WirelessProfile{
					ProfileName: "SecureProfile",
					IEEE8021x: &config.IEEE8021x{
						AuthenticationProtocol: 2,
						Username:               "user",
						Password:               "secret",
					},
				},
				res: models.IEEE8021xSettings{
					ElementName:            "SecureProfile",
					InstanceID:             "Intel(r) AMT:IEEE 802.1x Settings SecureProfile",
					AuthenticationProtocol: models.AuthenticationProtocol(2),
					Username:               "user",
					Password:               "secret",
				},
			},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				res := toIEEE8021xSettingsRequest(tc.profile)
				require.Equal(t, tc.res, res)
			})
		}
	})

	t.Run("wifiSettingToConfig", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			setting wifi.WiFiEndpointSettingsResponse
			res     config.WirelessProfile
		}{
			{
				name: "success",
				setting: wifi.WiFiEndpointSettingsResponse{
					ElementName:          "ProfileB",
					SSID:                 "SSID-B",
					AuthenticationMethod: wifi.AuthenticationMethodWPAIEEE8021x,
					EncryptionMethod:     wifi.EncryptionMethodTKIP,
					Priority:             3,
				},
				res: config.WirelessProfile{
					ProfileName:          "ProfileB",
					SSID:                 "SSID-B",
					AuthenticationMethod: "WPAIEEE8021x",
					EncryptionMethod:     "TKIP",
					Priority:             3,
				},
			},
		}

		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				res := wifiSettingToConfig(tc.setting)
				require.Equal(t, tc.res, res)
			})
		}
	})
}

func TestWiFiProfileFindExistingHandles(t *testing.T) {
	t.Parallel()

	certs := wifiProfileTestCertificates()

	tests := []struct {
		name       string
		finder     credentialHandleFinder
		credential string
		handle     string
		found      bool
	}{
		{
			name:       "private key found",
			finder:     findExistingPrivateKeyHandle,
			credential: "private-key",
			handle:     "pk-handle",
			found:      true,
		},
		{
			name:       "private key missing",
			finder:     findExistingPrivateKeyHandle,
			credential: "missing-private",
			handle:     "",
			found:      false,
		},
		{
			name:       "client cert found",
			finder:     findExistingClientCertHandle,
			credential: "client-cert",
			handle:     "client-handle",
			found:      true,
		},
		{
			name:       "client cert missing",
			finder:     findExistingClientCertHandle,
			credential: "ca-cert",
			handle:     "",
			found:      false,
		},
		{
			name:       "trusted root cert found",
			finder:     findExistingTrustedRootCertHandle,
			credential: "ca-cert",
			handle:     "root-handle",
			found:      true,
		},
		{
			name:       "trusted root cert missing",
			finder:     findExistingTrustedRootCertHandle,
			credential: "client-cert",
			handle:     "",
			found:      false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handle, found := tc.finder(certs, tc.credential)
			require.Equal(t, tc.found, found)
			require.Equal(t, tc.handle, handle)
		})
	}
}

func TestWiFiProfileResolveOrAddCredentialHandle(t *testing.T) {
	t.Parallel()

	baseCerts := wifiProfileTestCertificates()
	errGeneral := errors.New("general error")
	errAlreadyExists := errors.New("ALREADY EXISTS")

	refreshed := wsman.Certificates{
		PublicKeyCertificateResponse: publickey.RefinedPullResponse{
			PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{{
				InstanceID:             testNewHandle,
				X509Certificate:        "new-client",
				TrustedRootCertificate: false,
			}},
		},
	}

	tests := []struct {
		name       string
		credential string
		certs      wsman.Certificates
		find       credentialHandleFinder
		add        credentialHandleAdder
		refresh    certsRefresher
		handle     string
		resCerts   wsman.Certificates
		added      bool
		err        error
	}{
		{
			name:       "empty credential",
			credential: "",
			certs:      baseCerts,
			find:       findExistingClientCertHandle,
			add: func(_ string) (string, error) {
				return "", nil
			},
			refresh: func() (wsman.Certificates, error) {
				return wsman.Certificates{}, nil
			},
			handle:   "",
			resCerts: baseCerts,
			added:    false,
			err:      nil,
		},
		{
			name:       "existing handle is returned",
			credential: "client-cert",
			certs:      baseCerts,
			find:       findExistingClientCertHandle,
			add: func(_ string) (string, error) {
				return "", errors.New("unexpected add call")
			},
			refresh: func() (wsman.Certificates, error) {
				return wsman.Certificates{}, errors.New("unexpected refresh call")
			},
			handle:   "client-handle",
			resCerts: baseCerts,
			added:    false,
			err:      nil,
		},
		{
			name:       "add succeeds",
			credential: "new-client",
			certs:      baseCerts,
			find:       findExistingClientCertHandle,
			add: func(_ string) (string, error) {
				return testNewHandle, nil
			},
			refresh: func() (wsman.Certificates, error) {
				return wsman.Certificates{}, errors.New("unexpected refresh call")
			},
			handle:   testNewHandle,
			resCerts: baseCerts,
			added:    true,
			err:      nil,
		},
		{
			name:       "non already exists add error",
			credential: "new-client",
			certs:      baseCerts,
			find:       findExistingClientCertHandle,
			add: func(_ string) (string, error) {
				return "", errGeneral
			},
			refresh: func() (wsman.Certificates, error) {
				return wsman.Certificates{}, nil
			},
			handle:   "",
			resCerts: baseCerts,
			added:    false,
			err:      errGeneral,
		},
		{
			name:       "already exists refresh resolves",
			credential: "new-client",
			certs:      baseCerts,
			find:       findExistingClientCertHandle,
			add: func(_ string) (string, error) {
				return "", errAlreadyExists
			},
			refresh: func() (wsman.Certificates, error) {
				return refreshed, nil
			},
			handle:   testNewHandle,
			resCerts: refreshed,
			added:    false,
			err:      nil,
		},
		{
			name:       "already exists refresh fails",
			credential: "new-client",
			certs:      baseCerts,
			find:       findExistingClientCertHandle,
			add: func(_ string) (string, error) {
				return "", errAlreadyExists
			},
			refresh: func() (wsman.Certificates, error) {
				return wsman.Certificates{}, errGeneral
			},
			handle:   "",
			resCerts: wsman.Certificates{},
			added:    false,
			err:      errGeneral,
		},
		{
			name:       "already exists refresh missing handle",
			credential: "new-client",
			certs:      baseCerts,
			find:       findExistingClientCertHandle,
			add: func(_ string) (string, error) {
				return "", errAlreadyExists
			},
			refresh: func() (wsman.Certificates, error) {
				return baseCerts, nil
			},
			handle:   "",
			resCerts: baseCerts,
			added:    false,
			err:      errAlreadyExists,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handle, certs, added, err := resolveOrAddCredentialHandle(tc.certs, tc.credential, tc.find, tc.add, tc.refresh)
			require.Equal(t, tc.handle, handle)
			require.Equal(t, tc.resCerts, certs)
			require.Equal(t, tc.added, added)

			if tc.err != nil {
				require.IsType(t, tc.err, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFindWirelessSettingByProfileName(t *testing.T) {
	t.Parallel()

	settings := []wifi.WiFiEndpointSettingsResponse{
		{InstanceID: "", ElementName: "ignore-empty"},
		{InstanceID: instanceIDPrefixUserSettings + " Profile", ElementName: "ignored-user-setting"},
		{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Home", ElementName: "Home"},
	}

	res, found := findWirelessSettingByProfileName(settings, "Home")
	require.True(t, found)
	require.Equal(t, "Intel(r) AMT:WiFi Endpoint Settings Home", res.InstanceID)

	_, found = findWirelessSettingByProfileName(settings, "home")
	require.False(t, found)

	_, found = findWirelessSettingByProfileName(settings, "office")
	require.False(t, found)
}

func TestFindWirelessSettingByPriority(t *testing.T) {
	t.Parallel()

	settings := []wifi.WiFiEndpointSettingsResponse{
		{InstanceID: "", ElementName: "ignore-empty", Priority: 1},
		{InstanceID: instanceIDPrefixUserSettings + " Profile", ElementName: "ignored-user-setting", Priority: 1},
		{InstanceID: "Intel(r) AMT:WiFi Endpoint Settings Home", ElementName: "Home", Priority: 1},
	}

	res, found := findWirelessSettingByPriority(settings, 1)
	require.True(t, found)
	require.Equal(t, "Intel(r) AMT:WiFi Endpoint Settings Home", res.InstanceID)

	_, found = findWirelessSettingByPriority(settings, 2)
	require.False(t, found)
}

func TestWirelessProfileAlreadyExists(t *testing.T) {
	t.Parallel()

	err := wirelessProfileAlreadyExists("Home")
	require.Error(t, err)
	require.IsType(t, repoerrors.NotUniqueError{}, err)
}

func TestWirelessProfilePriorityAlreadyExists(t *testing.T) {
	t.Parallel()

	err := wirelessProfilePriorityAlreadyExists(1)
	require.Error(t, err)
	require.IsType(t, repoerrors.NotUniqueError{}, err)
}

func TestWiFiProfileIndexIEEE8021xSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		settings      []cimIEEE8021x.IEEE8021xSettingsResponse
		expectedNames map[string]string
	}{
		{
			name: "skips empty instance id",
			settings: []cimIEEE8021x.IEEE8021xSettingsResponse{
				{InstanceID: "", ElementName: "skip"},
				{InstanceID: "id-1", ElementName: "keep"},
			},
			expectedNames: map[string]string{"id-1": "keep"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			indexed := indexIEEE8021xSettings(tc.settings)
			require.Len(t, indexed, len(tc.expectedNames))

			for id, expectedName := range tc.expectedNames {
				require.Equal(t, expectedName, indexed[id].ElementName)
			}
		})
	}
}

func TestWiFiProfileDependencyReferencesForWiFi8021x(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		dependency      concrete.ConcreteDependency
		expectedFound   bool
		expectedWiFiURI string
		expectedIEEEURI string
	}{
		{
			name: "forward mapping",
			dependency: concrete.ConcreteDependency{
				Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings"}},
				Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings"}},
			},
			expectedFound:   true,
			expectedWiFiURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings",
			expectedIEEEURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings",
		},
		{
			name: "reverse mapping",
			dependency: concrete.ConcreteDependency{
				Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings"}},
				Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings"}},
			},
			expectedFound:   true,
			expectedWiFiURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings",
			expectedIEEEURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings",
		},
		{
			name: "no match",
			dependency: concrete.ConcreteDependency{
				Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ComputerSystem"}},
				Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings"}},
			},
			expectedFound: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wifiRef, ieeeRef, found := dependencyReferencesForWiFi8021x(tc.dependency)
			require.Equal(t, tc.expectedFound, found)

			if !tc.expectedFound {
				return
			}

			require.Equal(t, tc.expectedWiFiURI, wifiRef.ReferenceParameters.ResourceURI)
			require.Equal(t, tc.expectedIEEEURI, ieeeRef.ReferenceParameters.ResourceURI)
		})
	}
}

func TestWiFiProfileAssociationReferenceInstanceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		reference models.AssociationReference
		expected  string
		found     bool
	}{
		{
			name:      "found",
			reference: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "id-1"}}}}},
			expected:  "id-1",
			found:     true,
		},
		{
			name:      "empty instance id",
			reference: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: ""}}}}},
			expected:  "",
			found:     false,
		},
		{
			name:      "missing selector",
			reference: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "Name", Text: "x"}}}}},
			expected:  "",
			found:     false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id, ok := associationReferenceInstanceID(tc.reference)
			require.Equal(t, tc.found, ok)
			require.Equal(t, tc.expected, id)
		})
	}
}

func TestWiFiProfileMapAssociatedIEEE8021xByWiFiID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dependencies []concrete.ConcreteDependency
		expected     map[string]string
	}{
		{
			name: "skips incomplete references",
			dependencies: []concrete.ConcreteDependency{
				{
					Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "wifi-1"}}}}},
					Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "ieee-1"}}}}},
				},
				{
					Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "ieee-2"}}}}},
					Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "wifi-2"}}}}},
				},
				{
					Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_WiFiEndpointSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: ""}}}}},
					Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings", SelectorSet: models.SelectorNoNamespace{Selectors: []models.SelectorResponse{{Name: "InstanceID", Text: "ieee-skip"}}}}},
				},
				{
					Antecedent: models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ComputerSystem"}},
					Dependent:  models.AssociationReference{ReferenceParameters: models.ReferenceParametersNoNamespace{ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings"}},
				},
			},
			expected: map[string]string{
				"wifi-1": "ieee-1",
				"wifi-2": "ieee-2",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped := mapAssociatedIEEE8021xByWiFiID(tc.dependencies)
			require.Equal(t, tc.expected, mapped)
		})
	}
}

func TestWiFiProfileFindAssociatedIEEE8021xSettings(t *testing.T) {
	t.Parallel()

	ieeeByID := map[string]cimIEEE8021x.IEEE8021xSettingsResponse{
		"assoc-id": {InstanceID: "assoc-id", Username: "assoc-user"},
		"Intel(r) AMT:IEEE 802.1x Settings CorpFallback": {InstanceID: "Intel(r) AMT:IEEE 802.1x Settings CorpFallback", Username: "fallback-user"},
	}
	ieeeByName := map[string]cimIEEE8021x.IEEE8021xSettingsResponse{
		"corpname": {ElementName: "CorpName", Username: "name-user"},
	}

	tests := []struct {
		name          string
		setting       wifi.WiFiEndpointSettingsResponse
		associatedMap map[string]string
		byID          map[string]cimIEEE8021x.IEEE8021xSettingsResponse
		byName        map[string]cimIEEE8021x.IEEE8021xSettingsResponse
		expectedUser  string
		found         bool
	}{
		{
			name:          "direct association match",
			setting:       wifi.WiFiEndpointSettingsResponse{InstanceID: "wifi-1", ElementName: "Ignored"},
			associatedMap: map[string]string{"wifi-1": "assoc-id"},
			byID:          ieeeByID,
			byName:        ieeeByName,
			expectedUser:  "assoc-user",
			found:         true,
		},
		{
			name:          "fallback by generated instance id",
			setting:       wifi.WiFiEndpointSettingsResponse{ElementName: "CorpFallback"},
			associatedMap: map[string]string{},
			byID:          ieeeByID,
			byName:        ieeeByName,
			expectedUser:  "fallback-user",
			found:         true,
		},
		{
			name:          "fallback by normalized profile name",
			setting:       wifi.WiFiEndpointSettingsResponse{ElementName: "  CorpName  "},
			associatedMap: map[string]string{},
			byID:          map[string]cimIEEE8021x.IEEE8021xSettingsResponse{},
			byName:        ieeeByName,
			expectedUser:  "name-user",
			found:         true,
		},
		{
			name:          "no element name means not found",
			setting:       wifi.WiFiEndpointSettingsResponse{ElementName: ""},
			associatedMap: map[string]string{},
			byID:          map[string]cimIEEE8021x.IEEE8021xSettingsResponse{},
			byName:        map[string]cimIEEE8021x.IEEE8021xSettingsResponse{},
			expectedUser:  "",
			found:         false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, found := findAssociatedIEEE8021xSettings(tc.setting, tc.associatedMap, tc.byID, tc.byName)
			require.Equal(t, tc.found, found)
			require.Equal(t, tc.expectedUser, res.Username)
		})
	}
}
