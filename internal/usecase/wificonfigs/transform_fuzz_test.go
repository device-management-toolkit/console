package wificonfigs

import (
	"reflect"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func FuzzWirelessConfigTransforms(f *testing.F) {
	seedInputs := []struct {
		profileName      string
		ssid             string
		pskPassphrase    string
		entityLinkPolicy string
		ieeeName         string
		tenantID         string
		version          string
		authMethod       int
		encryptionMethod int
		pskValue         int
		linkA            int
		linkB            int
		setIEEEPtr       bool
		useNilLinkPolicy bool
	}{
		{"wifi-1", "ssid", "P@ssw0rd", "1,2,3", "ieee-1", "tenant-1", "1.0.0", 6, 4, 1, 1, 2, true, false},
		{"", "", "", "", "", "", "", 0, 0, 0, 0, 0, false, true},
		{"wifi_日本", "ssid/🙂", "päss\x00秘密", "-1,999999999999,foo,2", "ieee/特殊", "tenant/日本", "v1\n2", 7, 3, -1, -1, 999999999, true, false},
		{strings.Repeat("w", 2048), strings.Repeat("s", 1024), strings.Repeat("p", 4096), strings.Repeat("9,", 4096), strings.Repeat("i", 2048), strings.Repeat("t", 2048), strings.Repeat("v", 1024), -999999, 999999, -999999, -2147483648, 2147483647, false, false},
	}

	for i := range seedInputs {
		f.Add(seedInputs[i].profileName, seedInputs[i].ssid, seedInputs[i].pskPassphrase, seedInputs[i].entityLinkPolicy, seedInputs[i].ieeeName, seedInputs[i].tenantID, seedInputs[i].version, seedInputs[i].authMethod, seedInputs[i].encryptionMethod, seedInputs[i].pskValue, seedInputs[i].linkA, seedInputs[i].linkB, seedInputs[i].setIEEEPtr, seedInputs[i].useNilLinkPolicy)
	}

	uc := &UseCase{log: logger.New("error"), safeRequirements: mocks.MockCrypto{}}

	f.Fuzz(func(t *testing.T, profileName, ssid, pskPassphrase, entityLinkPolicy, ieeeName, tenantID, version string, authMethod, encryptionMethod, pskValue, linkA, linkB int, setIEEEPtr, useNilLinkPolicy bool) {
		buildDTO := func() *dto.WirelessConfig {
			return &dto.WirelessConfig{
				ProfileName:          profileName,
				AuthenticationMethod: authMethod,
				EncryptionMethod:     encryptionMethod,
				SSID:                 ssid,
				PSKValue:             pskValue,
				PSKPassphrase:        pskPassphrase,
				LinkPolicy:           []int{linkA, linkB},
				TenantID:             tenantID,
				IEEE8021xProfileName: stringPtrWireless(ieeeName, setIEEEPtr),
				Version:              version,
			}
		}

		buildEntity := func() *entity.WirelessConfig {
			var linkPolicy *string
			if !useNilLinkPolicy {
				linkPolicy = stringPtrWireless(entityLinkPolicy, true)
			}

			authProtocol := authMethod
			wiredInterface := authMethod%2 == 0

			return &entity.WirelessConfig{
				ProfileName:            profileName,
				AuthenticationMethod:   authMethod,
				EncryptionMethod:       encryptionMethod,
				SSID:                   ssid,
				PSKValue:               pskValue,
				PSKPassphrase:          pskPassphrase,
				LinkPolicy:             linkPolicy,
				TenantID:               tenantID,
				IEEE8021xProfileName:   stringPtrWireless(ieeeName, setIEEEPtr),
				Version:                version,
				AuthenticationProtocol: intPtrWireless(authProtocol, setIEEEPtr && ieeeName != ""),
				WiredInterface:         boolPtrWireless(wiredInterface, setIEEEPtr && ieeeName != ""),
			}
		}

		firstEntity, firstErr := uc.dtoToEntity(buildDTO())
		secondEntity, secondErr := uc.dtoToEntity(buildDTO())

		if !reflect.DeepEqual(firstErr, secondErr) {
			t.Fatalf("dtoToEntity error mismatch")
		}

		if firstErr != nil {
			t.Fatalf("dtoToEntity returned unexpected error: %v", firstErr)
		}

		if !reflect.DeepEqual(firstEntity, secondEntity) {
			t.Fatalf("dtoToEntity result mismatch")
		}

		verifyWifiEntityToDTO(t, uc, useNilLinkPolicy, buildEntity)
	})
}

func verifyWifiEntityToDTO(t *testing.T, uc *UseCase, useNilLinkPolicy bool, buildEntity func() *entity.WirelessConfig) {
	t.Helper()

	firstDTO := uc.entityToDTO(buildEntity())
	secondDTO := uc.entityToDTO(buildEntity())

	if !reflect.DeepEqual(firstDTO, secondDTO) {
		t.Fatalf("entityToDTO result mismatch")
	}

	if useNilLinkPolicy && len(firstDTO.LinkPolicy) != 0 {
		t.Fatalf("entityToDTO expected empty link policy for nil input")
	}
}

func stringPtrWireless(value string, enabled bool) *string {
	if !enabled {
		return nil
	}

	copyValue := value

	return &copyValue
}

func intPtrWireless(value int, enabled bool) *int {
	if !enabled {
		return nil
	}

	copyValue := value

	return &copyValue
}

func boolPtrWireless(value, enabled bool) *bool {
	if !enabled {
		return nil
	}

	copyValue := value

	return &copyValue
}
