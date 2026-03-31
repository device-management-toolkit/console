package ieee8021xconfigs

import (
	"reflect"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func FuzzIEEE8021xConfigTransforms(f *testing.F) {
	seedInputs := []struct {
		profileName   string
		tenantID      string
		version       string
		authProtocol  int
		pxeTimeout    int
		useNilTimeout bool
		wired         bool
	}{
		{"ieee-1", "tenant-1", "1.0.0", 2, 60, false, true},
		{"", "", "", 0, 0, true, false},
		{"ieee_日本", "tenant/日本", "v1\n2", 999999, -1, false, true},
		{strings.Repeat("i", 2048), strings.Repeat("t", 2048), strings.Repeat("v", 1024), -999999, 8640000, false, false},
	}

	for _, input := range seedInputs {
		f.Add(input.profileName, input.tenantID, input.version, input.authProtocol, input.pxeTimeout, input.useNilTimeout, input.wired)
	}

	uc := &UseCase{}

	f.Fuzz(func(t *testing.T, profileName, tenantID, version string, authProtocol, pxeTimeout int, useNilTimeout, wired bool) {
		buildDTO := func() *dto.IEEE8021xConfig {
			return &dto.IEEE8021xConfig{
				ProfileName:            profileName,
				AuthenticationProtocol: authProtocol,
				PXETimeout:             intPtrIEEE(pxeTimeout, !useNilTimeout),
				WiredInterface:         wired,
				TenantID:               tenantID,
				Version:                version,
			}
		}

		buildEntity := func() *entity.IEEE8021xConfig {
			return &entity.IEEE8021xConfig{
				ProfileName:            profileName,
				AuthenticationProtocol: authProtocol,
				PXETimeout:             intPtrIEEE(pxeTimeout, !useNilTimeout),
				WiredInterface:         wired,
				TenantID:               tenantID,
				Version:                version,
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

func intPtrIEEE(value int, enabled bool) *int {
	if !enabled {
		return nil
	}

	copyValue := value

	return &copyValue
}
