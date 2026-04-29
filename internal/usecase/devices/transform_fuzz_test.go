package devices

import (
	"reflect"
	"strings"
	"testing"
	"time"

	wsmanconfig "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const fuzzEncrypted = "encrypted"

func FuzzDeviceTransforms(f *testing.F) {
	seedInputs := []struct {
		guid             string
		dtoTagsRaw       string
		entityTagsRaw    string
		password         string
		mpsPassword      string
		mebxPassword     string
		certHash         string
		tenantID         string
		username         string
		useNilTags       bool
		useTime          bool
		connectionStatus bool
		useTLS           bool
		allowSelfSigned  bool
		timestamp        int64
	}{
		{"ABC-123", "alpha|beta", "alpha,beta", "P@ssw0rd", "mps-pass", "mebx-pass", "hash", "tenant-1", "admin", false, true, true, true, false, 1700000000},
		{"", "", "", "", "", "", "", "", "", true, false, false, false, false, 0},
		{"UPPER-🙂", "a|\x00|日本|🙂", "a,\x00,日本,🙂", "päss\x00秘密", "пароль", "🔐mebx", "hash/with:special", "tenant/日本", "user\nname", false, true, true, false, true, -2208988800},
		{strings.Repeat("G", 1024), strings.Repeat("tag|", 4096), strings.Repeat("tag,", 4096), strings.Repeat("p", 4096), strings.Repeat("m", 4096), strings.Repeat("x", 8192), strings.Repeat("c", 4096), strings.Repeat("t", 2048), strings.Repeat("u", 1024), false, true, false, true, true, 253402300799},
	}

	for i := range seedInputs {
		f.Add(seedInputs[i].guid, seedInputs[i].dtoTagsRaw, seedInputs[i].entityTagsRaw, seedInputs[i].password, seedInputs[i].mpsPassword, seedInputs[i].mebxPassword, seedInputs[i].certHash, seedInputs[i].tenantID, seedInputs[i].username, seedInputs[i].useNilTags, seedInputs[i].useTime, seedInputs[i].connectionStatus, seedInputs[i].useTLS, seedInputs[i].allowSelfSigned, seedInputs[i].timestamp)
	}

	uc := &UseCase{log: logger.New("error"), safeRequirements: fuzzCryptorDevice{}}

	f.Fuzz(func(t *testing.T, guid, dtoTagsRaw, entityTagsRaw, password, mpsPassword, mebxPassword, certHash, tenantID, username string, useNilTags, useTime, connectionStatus, useTLS, allowSelfSigned bool, timestamp int64) {
		buildTimePtr := func() *time.Time {
			if !useTime {
				return nil
			}

			value := time.Unix(timestamp, 0).UTC()

			return &value
		}

		buildDTO := func() *dto.Device {
			tags := buildDeviceTags(dtoTagsRaw, useNilTags)

			return &dto.Device{
				ConnectionStatus: connectionStatus,
				GUID:             guid,
				Tags:             tags,
				TenantID:         tenantID,
				Username:         username,
				Password:         password,
				MPSPassword:      mpsPassword,
				MEBXPassword:     mebxPassword,
				UseTLS:           useTLS,
				AllowSelfSigned:  allowSelfSigned,
				CertHash:         certHash,
				LastConnected:    buildTimePtr(),
				LastSeen:         buildTimePtr(),
				LastDisconnected: buildTimePtr(),
			}
		}

		buildEntity := func() *entity.Device {
			var entityTags []string
			if entityTagsRaw != "" {
				entityTags = strings.Split(entityTagsRaw, ",")
			}

			return &entity.Device{
				ConnectionStatus: connectionStatus,
				GUID:             guid,
				Tags:             entityTags,
				TenantID:         tenantID,
				Username:         username,
				Password:         password,
				MPSPassword:      stringPtrOrNilDevice(mpsPassword),
				MEBXPassword:     stringPtrOrNilDevice(mebxPassword),
				UseTLS:           useTLS,
				AllowSelfSigned:  allowSelfSigned,
				CertHash:         stringPtrOrNilDevice(certHash),
				LastConnected:    buildTimePtr(),
				LastSeen:         buildTimePtr(),
				LastDisconnected: buildTimePtr(),
			}
		}

		verifyDTOToEntity(t, uc, guid, certHash, buildDTO)
		verifyEntityToDTO(t, uc, buildEntity)
	})
}

func buildDeviceTags(dtoTagsRaw string, useNilTags bool) []string {
	if useNilTags {
		return nil
	}

	if dtoTagsRaw == "" {
		return []string{}
	}

	return strings.Split(dtoTagsRaw, "|")
}

func verifyDTOToEntity(t *testing.T, uc *UseCase, guid, certHash string, buildDTO func() *dto.Device) {
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

	if !strings.EqualFold(firstEntity.GUID, guid) {
		t.Fatalf("dtoToEntity did not lowercase GUID")
	}

	if certHash == "" && firstEntity.CertHash != nil {
		t.Fatalf("dtoToEntity expected nil cert hash")
	}
}

func verifyEntityToDTO(t *testing.T, uc *UseCase, buildEntity func() *entity.Device) {
	t.Helper()

	firstDTO := uc.entityToDTO(buildEntity())
	secondDTO := uc.entityToDTO(buildEntity())

	if !reflect.DeepEqual(firstDTO, secondDTO) {
		t.Fatalf("entityToDTO result mismatch")
	}
}

func stringPtrOrNilDevice(value string) *string {
	if value == "" {
		return nil
	}

	copyValue := value

	return &copyValue
}

type fuzzCryptorDevice struct{}

func (fuzzCryptorDevice) Encrypt(string) (string, error) {
	return fuzzEncrypted, nil
}

func (fuzzCryptorDevice) EncryptWithKey(string, string) (string, error) {
	return fuzzEncrypted, nil
}

func (fuzzCryptorDevice) GenerateKey() string {
	return "key"
}

func (fuzzCryptorDevice) Decrypt(string) (string, error) {
	return "decrypted", nil
}

func (fuzzCryptorDevice) ReadAndDecryptFile(string) (wsmanconfig.Configuration, error) {
	return wsmanconfig.Configuration{}, nil
}
