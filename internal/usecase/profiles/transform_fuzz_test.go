package profiles

import (
	"reflect"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
)

func FuzzProfileTransforms(f *testing.F) {
	seedInputs := []struct {
		profileName  string
		dtoTagsRaw   string
		entityTags   string
		amtPassword  string
		mebxPassword string
		ciraName     string
		ieeeName     string
		creationDate string
		userConsent  string
		tenantID     string
		version      string
		tlsMode      int
		useNilTags   bool
		setCIRAPtr   bool
		setIEEEPtr   bool
		dhcpEnabled  bool
	}{
		{"profile-1", "alpha|beta", "alpha,beta", "P@ssw0rd", "MebxP@ss", "cira-1", "ieee-1", "2021-07-01T00:00:00Z", "All", "tenant-1", "1", 1, false, true, true, true},
		{"", "", "", "", "", "", "", "", "", "", "", 0, true, false, false, false},
		{"プロファイル", "a|\x00|日本|🙂", "a,,日本,🙂", "päss\x00秘密", "🔐mēbx", "cira/特殊", "ieee-🙂", "not-a-date", "KVM", "tenant/日本", "v1\n2", 4, false, true, true, false},
		{strings.Repeat("p", 2048), strings.Repeat("tag|", 4096), strings.Repeat("tag,", 4096), strings.Repeat("a", 4096), strings.Repeat("m", 4096), strings.Repeat("c", 1024), strings.Repeat("i", 1024), strings.Repeat("2", 512), "None", strings.Repeat("t", 1024), strings.Repeat("v", 1024), -1, false, true, true, true},
	}

	for i := range seedInputs {
		f.Add(seedInputs[i].profileName, seedInputs[i].dtoTagsRaw, seedInputs[i].entityTags, seedInputs[i].amtPassword, seedInputs[i].mebxPassword, seedInputs[i].ciraName, seedInputs[i].ieeeName, seedInputs[i].creationDate, seedInputs[i].userConsent, seedInputs[i].tenantID, seedInputs[i].version, seedInputs[i].tlsMode, seedInputs[i].useNilTags, seedInputs[i].setCIRAPtr, seedInputs[i].setIEEEPtr, seedInputs[i].dhcpEnabled)
	}

	uc := &UseCase{safeRequirements: mocks.MockCrypto{}}

	f.Fuzz(func(t *testing.T, profileName, dtoTagsRaw, entityTags, amtPassword, mebxPassword, ciraName, ieeeName, creationDate, userConsent, tenantID, version string, tlsMode int, useNilTags, setCIRAPtr, setIEEEPtr, dhcpEnabled bool) {
		buildDTO := func() *dto.Profile {
			tags := buildProfileTags(dtoTagsRaw, useNilTags)

			return &dto.Profile{
				ProfileName:          profileName,
				AMTPassword:          amtPassword,
				CreationDate:         creationDate,
				Tags:                 tags,
				DHCPEnabled:          dhcpEnabled,
				TenantID:             tenantID,
				TLSMode:              tlsMode,
				UserConsent:          userConsent,
				MEBXPassword:         mebxPassword,
				Version:              version,
				CIRAConfigName:       stringPtrProfile(ciraName, setCIRAPtr),
				IEEE8021xProfileName: stringPtrProfile(ieeeName, setIEEEPtr),
			}
		}

		buildEntity := func() *entity.Profile {
			authProtocol := tlsMode
			wiredInterface := dhcpEnabled

			return &entity.Profile{
				ProfileName:                profileName,
				CreationDate:               creationDate,
				Tags:                       entityTags,
				DHCPEnabled:                dhcpEnabled,
				TenantID:                   tenantID,
				TLSMode:                    tlsMode,
				UserConsent:                userConsent,
				GenerateRandomPassword:     false,
				GenerateRandomMEBxPassword: false,
				Version:                    version,
				CIRAConfigName:             stringPtrProfile(ciraName, setCIRAPtr),
				IEEE8021xProfileName:       stringPtrProfile(ieeeName, setIEEEPtr),
				AuthenticationProtocol:     intPtrProfile(authProtocol, setIEEEPtr && ieeeName != ""),
				WiredInterface:             boolPtrProfile(wiredInterface, setIEEEPtr && ieeeName != ""),
			}
		}

		verifyProfileDTOToEntity(t, uc, dtoTagsRaw, useNilTags, buildDTO)
		verifyProfileEntityToDTO(t, uc, buildEntity)
	})
}

func buildProfileTags(dtoTagsRaw string, useNilTags bool) []string {
	if useNilTags {
		return nil
	}

	if dtoTagsRaw == "" {
		return []string{}
	}

	return strings.Split(dtoTagsRaw, "|")
}

func verifyProfileDTOToEntity(t *testing.T, uc *UseCase, dtoTagsRaw string, useNilTags bool, buildDTO func() *dto.Profile) {
	t.Helper()

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

	if firstEntity.Tags != strings.Join(splitTagsProfile(dtoTagsRaw, useNilTags), ", ") {
		t.Fatalf("dtoToEntity tag join mismatch")
	}
}

func verifyProfileEntityToDTO(t *testing.T, uc *UseCase, buildEntity func() *entity.Profile) {
	t.Helper()

	firstDTO := uc.entityToDTO(buildEntity())
	secondDTO := uc.entityToDTO(buildEntity())

	if !reflect.DeepEqual(firstDTO, secondDTO) {
		t.Fatalf("entityToDTO result mismatch")
	}
}

func splitTagsProfile(raw string, useNilTags bool) []string {
	if useNilTags {
		return []string{}
	}

	if raw == "" {
		return []string{}
	}

	return strings.Split(raw, "|")
}

func stringPtrProfile(value string, enabled bool) *string {
	if !enabled {
		return nil
	}

	copyValue := value

	return &copyValue
}

func intPtrProfile(value int, enabled bool) *int {
	if !enabled {
		return nil
	}

	copyValue := value

	return &copyValue
}

func boolPtrProfile(value, enabled bool) *bool {
	if !enabled {
		return nil
	}

	copyValue := value

	return &copyValue
}
