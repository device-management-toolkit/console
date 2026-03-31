package ciraconfigs

import (
	"reflect"
	"strings"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func FuzzCIRAConfigTransforms(f *testing.F) {
	seedInputs := []struct {
		configName          string
		address             string
		password            string
		commonName          string
		rootCertificate     string
		proxyDetails        string
		tenantID            string
		version             string
		port                int
		serverAddressFormat int
		authMethod          int
		generateRandom      bool
	}{
		{"cira-1", "https://example.com", "P@ssw0rd", "example.com", "root-cert", "http://proxy", "tenant-1", "1.0.0", 4433, 201, 2, false},
		{"", "", "", "", "", "", "", "", 0, 0, 0, false},
		{"cira_日本", "https://例え.テスト", "päss\x00秘密🔐", "例え.テスト", strings.Repeat("r", 2048), "socks5://代理", "tenant/日本", "v1\n2", -1, -4, 99, true},
		{strings.Repeat("c", 2048), strings.Repeat("a", 4096), strings.Repeat("p", 4096), strings.Repeat("n", 1024), strings.Repeat("r", 4096), strings.Repeat("x", 4096), strings.Repeat("t", 1024), strings.Repeat("v", 1024), 65535, 999999, -999999, false},
	}

	for i := range seedInputs {
		f.Add(seedInputs[i].configName, seedInputs[i].address, seedInputs[i].password, seedInputs[i].commonName, seedInputs[i].rootCertificate, seedInputs[i].proxyDetails, seedInputs[i].tenantID, seedInputs[i].version, seedInputs[i].port, seedInputs[i].serverAddressFormat, seedInputs[i].authMethod, seedInputs[i].generateRandom)
	}

	uc := &UseCase{log: logger.New("error"), safeRequirements: mocks.MockCrypto{}}

	f.Fuzz(func(t *testing.T, configName, address, password, commonName, rootCertificate, proxyDetails, tenantID, version string, port, serverAddressFormat, authMethod int, generateRandom bool) {
		buildDTO := func() *dto.CIRAConfig {
			return &dto.CIRAConfig{
				ConfigName:             configName,
				MPSAddress:             address,
				MPSPort:                port,
				Username:               commonName,
				Password:               password,
				CommonName:             commonName,
				ServerAddressFormat:    serverAddressFormat,
				AuthMethod:             authMethod,
				MPSRootCertificate:     rootCertificate,
				ProxyDetails:           proxyDetails,
				TenantID:               tenantID,
				GenerateRandomPassword: generateRandom,
				Version:                version,
			}
		}

		buildEntity := func() *entity.CIRAConfig {
			return &entity.CIRAConfig{
				ConfigName:             configName,
				MPSAddress:             address,
				MPSPort:                port,
				Username:               commonName,
				Password:               password,
				CommonName:             commonName,
				ServerAddressFormat:    serverAddressFormat,
				AuthMethod:             authMethod,
				MPSRootCertificate:     rootCertificate,
				ProxyDetails:           proxyDetails,
				TenantID:               tenantID,
				GenerateRandomPassword: generateRandom,
				Version:                version,
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

		firstDTO := uc.entityToDTO(buildEntity())
		secondDTO := uc.entityToDTO(buildEntity())

		if !reflect.DeepEqual(firstDTO, secondDTO) {
			t.Fatalf("entityToDTO result mismatch")
		}
	})
}
