package domains

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func FuzzDomainTransforms(f *testing.F) {
	seedInputs := []struct {
		profileName  string
		domainSuffix string
		cert         string
		password     string
		expiration   string
		tenantID     string
		version      string
	}{
		{"domain-1", "example.com", "cert", "P@ssw0rd", "2024-01-02T03:04:05Z", "tenant-1", "1.0.0"},
		{"", "", "", "", "", "", ""},
		{"domain_日本", "例え.テスト", strings.Repeat("c", 2048), "päss\x00秘密🔐", "2016-12-31T23:59:60Z", "tenant/日本", "v1\n2"},
		{strings.Repeat("p", 2048), strings.Repeat("d", 2048), strings.Repeat("c", 4096), strings.Repeat("x", 4096), "9999-12-31T23:59:59+14:00", strings.Repeat("t", 2048), strings.Repeat("v", 2048)},
		{"edge-domain", "example.org", "cert", "pw", "0000-01-01T00:00:00Z", "tenant-edge", "2.0.0"},
		{"bad-domain", "example.net", "cert", "pw", "not-a-time", "tenant-bad", "3.0.0"},
	}

	for _, input := range seedInputs {
		f.Add(input.profileName, input.domainSuffix, input.cert, input.password, input.expiration, input.tenantID, input.version)
	}

	uc := &UseCase{log: logger.New("error"), safeRequirements: mocks.MockCrypto{}}

	f.Fuzz(func(t *testing.T, profileName, domainSuffix, cert, password, expiration, tenantID, version string) {
		buildDTO := func() *dto.Domain {
			return &dto.Domain{
				ProfileName:                   profileName,
				DomainSuffix:                  domainSuffix,
				ProvisioningCert:              cert,
				ProvisioningCertPassword:      password,
				ProvisioningCertStorageFormat: "string",
				TenantID:                      tenantID,
				Version:                       version,
			}
		}

		buildEntity := func() *entity.Domain {
			return &entity.Domain{
				ProfileName:                   profileName,
				DomainSuffix:                  domainSuffix,
				ProvisioningCert:              cert,
				ProvisioningCertPassword:      password,
				ProvisioningCertStorageFormat: "string",
				ExpirationDate:                expiration,
				TenantID:                      tenantID,
				Version:                       version,
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

		verifyDomainEntityToDTO(t, uc, password, expiration, firstEntity, buildEntity)
	})
}

func verifyDomainEntityToDTO(t *testing.T, uc *UseCase, password, expiration string, firstEntity *entity.Domain, buildEntity func() *entity.Domain) {
	t.Helper()

	firstDTO := uc.entityToDTO(buildEntity())
	secondDTO := uc.entityToDTO(buildEntity())

	if !reflect.DeepEqual(firstDTO, secondDTO) {
		t.Fatalf("entityToDTO result mismatch")
	}

	expectedTime, err := time.Parse(time.RFC3339, expiration)
	if err == nil {
		if !firstDTO.ExpirationDate.Equal(expectedTime) {
			t.Fatalf("entityToDTO expiration parse mismatch")
		}
	} else if !firstDTO.ExpirationDate.IsZero() {
		t.Fatalf("entityToDTO expected zero expiration for invalid timestamp %q", expiration)
	}

	if strings.Contains(password, "\x00") && firstEntity.ProvisioningCertPassword == password {
		t.Fatalf("dtoToEntity did not transform password input")
	}
}
