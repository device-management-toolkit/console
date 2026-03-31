package dto

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func FuzzDeviceJSONProcessing(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"guid":"ABC-123","username":"admin","password":"P@ssw0rd","mpspassword":"秘密","mebxpassword":"🔐pw","tags":["a","b"],"lastConnected":"2024-01-02T03:04:05Z"}`,
		`{"guid":null,"tags":null,"lastConnected":null}`,
		`{"guid":123,"username":true,"tags":"not-an-array","lastConnected":"not-a-time"}`,
		fmt.Sprintf(`{"guid":"huge","tags":[%s]}`, quotedArray("tag", 4096)),
		fmt.Sprintf(`{"guid":"nested","junk":%s}`, nestedJSONObject()),
		`{"guid":"unicode","username":"用戶🙂","password":"päss\u0000秘密","mpspassword":"пароль","lastConnected":"9999-12-31T23:59:59+14:00"}`,
		`{"guid":"year-zero","lastConnected":"0000-01-01T00:00:00Z"}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	v := validator.New()
	v.SetTagName("binding")

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONAndValidate(t, payload, func() *Device { return &Device{} }, func(value *Device) error {
			return v.Struct(value)
		})
	})
}

func FuzzProfileJSONProcessing(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"profileName":"profile-1","activation":"acmactivate","amtPassword":"P@ssw0rd!","mebxPassword":"MebxP@ss!","generateRandomPassword":false,"generateRandomMEBxPassword":false,"dhcpEnabled":true,"wifiConfigs":[{"priority":1,"profileName":"wifi-1","profileProfileName":"profile-1"}],"ciraConfigName":"cira-1","ieee8021xProfileName":"ieee-1","tags":["a","b"]}`,
		`{"profileName":"partial","ciraConfigName":null,"ieee8021xProfileName":null,"tags":null,"wifiConfigs":null}`,
		`{"profileName":123,"tlsMode":"bad","dhcpEnabled":"false","wifiConfigs":"not-an-array"}`,
		fmt.Sprintf(`{"profileName":"huge","activation":"ccmactivate","generateRandomPassword":true,"generateRandomMEBxPassword":true,"tags":[%s],"wifiConfigs":[%s]}`, quotedArray("tag", 4096), repeatedJSON(`{"priority":1,"profileName":"wifi","profileProfileName":"profile"}`, 512)),
		fmt.Sprintf(`{"profileName":"nested","activation":"ccmactivate","generateRandomPassword":true,"generateRandomMEBxPassword":true,"junk":%s}`, nestedJSONObject()),
		`{"profileName":"contradictory","activation":"ccmactivate","generateRandomPassword":false,"amtPassword":"","generateRandomMEBxPassword":false,"mebxPassword":"","ciraConfigName":"cira","tlsMode":1,"dhcpEnabled":false,"wifiConfigs":[{"priority":1,"profileName":"wifi-1","profileProfileName":"profile-1"}]}`,
		`{"profileName":"unicode","activation":"acmactivate","amtPassword":"päss🙂秘密!","mebxPassword":"пароль🔐!","generateRandomPassword":false,"generateRandomMEBxPassword":false,"tags":["日本","\u0000","🙂"]}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	v := newProfileValidatorForFuzz()

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONAndValidate(t, payload, func() *Profile { return &Profile{} }, func(value *Profile) error {
			return v.Struct(value)
		})
	})
}

func FuzzDomainJSONProcessing(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"profileName":"domain-1","domainSuffix":"example.com","provisioningCert":"cert","provisioningCertStorageFormat":"string","provisioningCertPassword":"P@ssw0rd","expirationDate":"2024-01-02T03:04:05Z"}`,
		`{"profileName":null,"provisioningCert":null,"expirationDate":null}`,
		`{"profileName":123,"provisioningCertPassword":false,"expirationDate":"not-a-time"}`,
		fmt.Sprintf(`{"profileName":"nested","domainSuffix":"example.com","provisioningCert":"cert","provisioningCertStorageFormat":"string","provisioningCertPassword":"pw","junk":%s}`, nestedJSONObject()),
		`{"profileName":"unicode_日本","domainSuffix":"例え.テスト","provisioningCert":"cert","provisioningCertStorageFormat":"string","provisioningCertPassword":"päss\u0000秘密🔐","expirationDate":"2016-12-31T23:59:60Z"}`,
		`{"profileName":"year-zero","domainSuffix":"example.org","provisioningCert":"cert","provisioningCertStorageFormat":"string","provisioningCertPassword":"pw","expirationDate":"0000-01-01T00:00:00Z"}`,
		`{"profileName":"year-max","domainSuffix":"example.org","provisioningCert":"cert","provisioningCertStorageFormat":"string","provisioningCertPassword":"pw","expirationDate":"9999-12-31T23:59:59+14:00"}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	v := validator.New()
	v.SetTagName("binding")
	_ = v.RegisterValidation("alphanumhyphenunderscore", ValidateAlphaNumHyphenUnderscore)

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONAndValidate(t, payload, func() *Domain { return &Domain{} }, func(value *Domain) error {
			return v.Struct(value)
		})
	})
}

func FuzzCIRAConfigJSONProcessing(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"configName":"cira-1","mpsServerAddress":"https://example.com","mpsPort":4433,"username":"admin","password":"P@ssw0rd","commonName":"example.com","serverAddressFormat":201,"authMethod":2,"mpsRootCertificate":"cert"}`,
		`{"configName":null,"mpsServerAddress":null,"password":null}`,
		`{"mpsPort":"4433","serverAddressFormat":"201","authMethod":"2"}`,
		fmt.Sprintf(`{"configName":"nested","mpsServerAddress":"https://example.com","mpsPort":4433,"username":"admin","password":"pw","commonName":"example.com","serverAddressFormat":201,"authMethod":2,"mpsRootCertificate":"cert","junk":%s}`, nestedJSONObject()),
		`{"configName":"unicode","mpsServerAddress":"https://例え.テスト","mpsPort":65535,"username":"用戶🙂","password":"päss\u0000秘密🔐","commonName":"例え.テスト","serverAddressFormat":999,"authMethod":99,"mpsRootCertificate":"cert"}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	v := validator.New()
	v.SetTagName("binding")
	_ = v.RegisterValidation("alphanumhyphenunderscore", ValidateAlphaNumHyphenUnderscore)

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONAndValidate(t, payload, func() *CIRAConfig { return &CIRAConfig{} }, func(value *CIRAConfig) error {
			return v.Struct(value)
		})
	})
}

func FuzzWirelessConfigJSONProcessing(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"profileName":"wifi-1","authenticationMethod":7,"encryptionMethod":4,"ssid":"ssid","pskPassphrase":"P@ssw0rd","linkPolicy":[1,2],"ieee8021xProfileName":"ieee-1"}`,
		`{"profileName":"wifi-null","linkPolicy":null,"ieee8021xProfileName":null,"ieee8021xProfileObject":null}`,
		`{"authenticationMethod":"7","linkPolicy":"not-an-array","pskValue":"bad"}`,
		fmt.Sprintf(`{"profileName":"huge","authenticationMethod":6,"encryptionMethod":4,"ssid":"ssid","pskPassphrase":"pw","linkPolicy":[%s]}`, intArray(4096)),
		fmt.Sprintf(`{"profileName":"nested","authenticationMethod":6,"encryptionMethod":4,"ssid":"ssid","pskPassphrase":"pw","linkPolicy":[1,2],"ieee8021xProfileObject":%s}`, nestedIEEE8021xObject(16)),
		`{"profileName":"contradictory","authenticationMethod":7,"encryptionMethod":3,"ssid":"ssid","pskPassphrase":"päss\u0000秘密","linkPolicy":[-1,999999999],"ieee8021xProfileName":null}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	v := validator.New()
	v.SetTagName("binding")
	_ = v.RegisterValidation("authforieee8021x", ValidateAuthandIEEE)
	_ = v.RegisterValidation("authProtocolValidator", AuthProtocolValidator)

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONAndValidate(t, payload, func() *WirelessConfig { return &WirelessConfig{} }, func(value *WirelessConfig) error {
			return v.Struct(value)
		})
	})
}

func FuzzIEEE8021xJSONProcessing(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"profileName":"ieee-1","authenticationProtocol":2,"pxeTimeout":60,"wiredInterface":true}`,
		`{"profileName":"nulls","pxeTimeout":null}`,
		`{"authenticationProtocol":"2","pxeTimeout":"60","wiredInterface":"false"}`,
		fmt.Sprintf(`{"profileName":"nested","authenticationProtocol":2,"pxeTimeout":60,"wiredInterface":true,"junk":%s}`, nestedJSONObject()),
		`{"profileName":"edge","authenticationProtocol":999,"pxeTimeout":86401,"wiredInterface":true}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	v := validator.New()
	v.SetTagName("binding")
	_ = v.RegisterValidation("authProtocolValidator", AuthProtocolValidator)

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONAndValidate(t, payload, func() *IEEE8021xConfig { return &IEEE8021xConfig{} }, func(value *IEEE8021xConfig) error {
			return v.Struct(value)
		})
	})
}

func FuzzProfileWiFiJSONProcessing(f *testing.F) {
	seedInputs := []string{
		`{}`,
		`{"priority":1,"profileName":"wifi-1","profileProfileName":"profile-1","tenantId":"tenant-1"}`,
		`{"priority":null,"profileName":null}`,
		`{"priority":"1","profileName":123}`,
		fmt.Sprintf(`{"priority":1,"profileName":"wifi-1","profileProfileName":"profile-1","junk":%s}`, nestedJSONObject()),
		`{"priority":999999999,"profileName":"wifi_日本🙂","profileProfileName":"profile/特殊","tenantId":"tenant/日本"}`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	v := validator.New()
	v.SetTagName("binding")

	f.Fuzz(func(t *testing.T, payload string) {
		fuzzJSONAndValidate(t, payload, func() *ProfileWiFiConfigs { return &ProfileWiFiConfigs{} }, func(value *ProfileWiFiConfigs) error {
			return v.Struct(value)
		})
	})
}

func fuzzJSONAndValidate[T any](t *testing.T, payload string, newValue func() *T, validate func(*T) error) {
	t.Helper()

	first := newValue()
	second := newValue()

	firstErr := json.Unmarshal([]byte(payload), first)
	secondErr := json.Unmarshal([]byte(payload), second)

	if (firstErr == nil) != (secondErr == nil) {
		t.Fatalf("json.Unmarshal error mismatch for payload %q: first=%v second=%v", payload, firstErr, secondErr)
	}

	if firstErr != nil {
		if firstErr.Error() != secondErr.Error() {
			t.Fatalf("json.Unmarshal error text mismatch for payload %q: first=%q second=%q", payload, firstErr.Error(), secondErr.Error())
		}

		return
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("json.Unmarshal result mismatch for payload %q", payload)
	}

	if validate == nil {
		return
	}

	firstValidationErr := validate(first)
	secondValidationErr := validate(second)

	if (firstValidationErr == nil) != (secondValidationErr == nil) {
		t.Fatalf("validation error mismatch for payload %q: first=%v second=%v", payload, firstValidationErr, secondValidationErr)
	}

	if firstValidationErr != nil && firstValidationErr.Error() != secondValidationErr.Error() {
		t.Fatalf("validation error text mismatch for payload %q: first=%q second=%q", payload, firstValidationErr.Error(), secondValidationErr.Error())
	}
}

func newProfileValidatorForFuzz() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	_ = v.RegisterValidation("genpasswordwone", ValidateAMTPassOrGenRan)
	_ = v.RegisterValidation("ciraortls", ValidateCIRAOrTLS)
	_ = v.RegisterValidation("wifidhcp", ValidateWiFiDHCP)

	return v
}

func quotedArray(value string, count int) string {
	items := make([]string, count)
	for index := range items {
		items[index] = fmt.Sprintf("%q", value)
	}

	return strings.Join(items, ",")
}

func intArray(count int) string {
	items := make([]string, count)
	for index := range items {
		items[index] = fmt.Sprintf("%d", index)
	}

	return strings.Join(items, ",")
}

func repeatedJSON(item string, count int) string {
	items := make([]string, count)
	for index := range items {
		items[index] = item
	}

	return strings.Join(items, ",")
}

const nestedJSONDepth = 32

func nestedJSONObject() string {
	result := `{"leaf":true}`
	for index := 0; index < nestedJSONDepth; index++ {
		result = fmt.Sprintf(`{"level%d":%s}`, index, result)
	}

	return result
}

func nestedIEEE8021xObject(depth int) string {
	result := `{"profileName":"ieee-1","authenticationProtocol":2,"pxeTimeout":60,"wiredInterface":true}`
	for index := 0; index < depth; index++ {
		result = fmt.Sprintf(`{"nested%d":%s}`, index, result)
	}

	return result
}
