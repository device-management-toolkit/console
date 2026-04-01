package profilewificonfigs

import (
	"reflect"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func FuzzProfileWiFiConfigTransforms(f *testing.F) {
	seedInputs := []struct {
		priority            int
		wirelessProfileName string
		profileName         string
		tenantID            string
	}{
		{1, "wireless-1", "profile-1", "tenant-1"},
		{0, "", "", ""},
		{-1, "wireless_日本🙂", "profile/特殊", "tenant/日本"},
		{999999999, strings.Repeat("w", 2048), strings.Repeat("p", 2048), strings.Repeat("t", 2048)},
	}

	for _, input := range seedInputs {
		f.Add(input.priority, input.wirelessProfileName, input.profileName, input.tenantID)
	}

	uc := &UseCase{}

	f.Fuzz(func(t *testing.T, priority int, wirelessProfileName, profileName, tenantID string) {
		buildDTO := func() *dto.ProfileWiFiConfigs {
			return &dto.ProfileWiFiConfigs{
				Priority:            priority,
				WirelessProfileName: wirelessProfileName,
				ProfileName:         profileName,
				TenantID:            tenantID,
			}
		}

		buildEntity := func() *entity.ProfileWiFiConfigs {
			return &entity.ProfileWiFiConfigs{
				Priority:            priority,
				WirelessProfileName: wirelessProfileName,
				ProfileName:         profileName,
				TenantID:            tenantID,
			}
		}

		firstEntity := uc.dtoToEntity(buildDTO())
		secondEntity := uc.dtoToEntity(buildDTO())

		if !reflect.DeepEqual(firstEntity, secondEntity) {
			t.Fatalf("dtoToEntity result mismatch")
		}

		firstDTO := uc.entityToDTO(buildEntity())
		secondDTO := uc.entityToDTO(buildEntity())

		if !reflect.DeepEqual(firstDTO, secondDTO) {
			t.Fatalf("entityToDTO result mismatch")
		}
	})
}
