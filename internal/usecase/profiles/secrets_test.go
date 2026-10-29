package profiles_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var errVault = errors.New("vault unavailable")

type fakeSecretStore struct {
	set     map[string]map[string]string
	deleted []string
	setErr  error
}

func (f *fakeSecretStore) SetObject(key string, data map[string]string) error {
	if f.setErr != nil {
		return f.setErr
	}

	f.set[key] = data

	return nil
}

func (f *fakeSecretStore) DeleteKeyValue(key string) error {
	f.deleted = append(f.deleted, key)

	return nil
}

type secretsFixture struct {
	uc    *profiles.UseCase
	repo  *mocks.MockProfilesRepository
	pwc   *mocks.MockProfileWiFiConfigsFeature
	store *fakeSecretStore
}

func newSecretsFixture(t *testing.T) secretsFixture {
	t.Helper()

	ctl := gomock.NewController(t)
	repo := mocks.NewMockProfilesRepository(ctl)
	pwc := mocks.NewMockProfileWiFiConfigsFeature(ctl)
	store := &fakeSecretStore{set: map[string]map[string]string{}}

	uc := profiles.New(repo, mocks.NewMockWiFiConfigsRepository(ctl), pwc, mocks.NewMockIEEE8021xConfigsFeature(ctl),
		logger.New("error"), mocks.NewMockDomainsFeature(ctl), mocks.NewMockCIRAConfigsRepository(ctl),
		mocks.MockCrypto{}, false, profiles.WithSecretStore(store))

	return secretsFixture{uc: uc, repo: repo, pwc: pwc, store: store}
}

func TestInsertWritesProfileSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   dto.Profile
		want    map[string]string
		deleted []string
	}{
		{
			name:  "static AMT password, random MEBx",
			input: dto.Profile{ProfileName: "p1", TenantID: "t1", Activation: dto.ActivationCCM, AMTPassword: "P@ssw0rd", GenerateRandomMEBxPassword: true},
			want:  map[string]string{"AMT_PASSWORD": "P@ssw0rd", "MEBX_PASSWORD": ""},
		},
		{
			name:  "static AMT and MEBx passwords",
			input: dto.Profile{ProfileName: "p1", TenantID: "t1", Activation: dto.ActivationACM, AMTPassword: "P@ssw0rd", MEBXPassword: "M3bx!pwd"},
			want:  map[string]string{"AMT_PASSWORD": "P@ssw0rd", "MEBX_PASSWORD": "M3bx!pwd"},
		},
		{
			name:    "both random, nothing to store",
			input:   dto.Profile{ProfileName: "p1", TenantID: "t1", Activation: dto.ActivationCCM, GenerateRandomPassword: true, GenerateRandomMEBxPassword: true},
			deleted: []string{"profiles/p1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newSecretsFixture(t)
			f.repo.EXPECT().Insert(context.Background(), gomock.Any()).Return("id", nil)
			f.repo.EXPECT().GetByName(context.Background(), "p1", "t1").Return(&entity.Profile{ProfileName: "p1", TenantID: "t1"}, nil)

			_, err := f.uc.Insert(context.Background(), &tc.input)
			require.NoError(t, err)

			if tc.want != nil {
				require.Equal(t, tc.want, f.store.set["profiles/p1"])
			} else {
				require.Empty(t, f.store.set)
			}

			require.Equal(t, tc.deleted, f.store.deleted)
		})
	}
}

func TestInsertRollsBackWhenSecretWriteFails(t *testing.T) {
	t.Parallel()

	f := newSecretsFixture(t)
	f.store.setErr = errVault

	f.repo.EXPECT().Insert(context.Background(), gomock.Any()).Return("id", nil)
	f.pwc.EXPECT().DeleteByProfileName(context.Background(), "p1", "t1").Return(nil)
	f.repo.EXPECT().Delete(context.Background(), "p1", "t1").Return(true, nil)

	res, err := f.uc.Insert(context.Background(), &dto.Profile{ProfileName: "p1", TenantID: "t1", Activation: dto.ActivationCCM, AMTPassword: "P@ssw0rd", GenerateRandomMEBxPassword: true})

	require.Nil(t, res)
	require.ErrorIs(t, err, errVault)
}

// The Vault write error still comes back to the caller when the rollback itself fails.
func TestInsertReportsVaultErrorWhenRollbackFails(t *testing.T) {
	t.Parallel()

	f := newSecretsFixture(t)
	f.store.setErr = errVault

	f.repo.EXPECT().Insert(context.Background(), gomock.Any()).Return("id", nil)
	f.pwc.EXPECT().DeleteByProfileName(context.Background(), "p1", "t1").Return(errors.New("rollback failed"))

	res, err := f.uc.Insert(context.Background(), &dto.Profile{ProfileName: "p1", TenantID: "t1", Activation: dto.ActivationCCM, AMTPassword: "P@ssw0rd", GenerateRandomMEBxPassword: true})

	require.Nil(t, res)
	require.ErrorIs(t, err, errVault)
}

func TestUpdateWritesProfileSecrets(t *testing.T) {
	t.Parallel()

	f := newSecretsFixture(t)
	f.repo.EXPECT().Update(context.Background(), gomock.Any()).Return(true, nil)
	f.pwc.EXPECT().DeleteByProfileName(context.Background(), "p1", "t1").Return(nil)
	f.repo.EXPECT().GetByName(context.Background(), "p1", "t1").Return(&entity.Profile{ProfileName: "p1", TenantID: "t1"}, nil)

	_, err := f.uc.Update(context.Background(), &dto.Profile{ProfileName: "p1", TenantID: "t1", Activation: dto.ActivationCCM, AMTPassword: "N3wP@ss", GenerateRandomMEBxPassword: true}, nil)

	require.NoError(t, err)
	require.Equal(t, map[string]string{"AMT_PASSWORD": "N3wP@ss", "MEBX_PASSWORD": ""}, f.store.set["profiles/p1"])
}

func TestDeleteRemovesProfileSecrets(t *testing.T) {
	t.Parallel()

	f := newSecretsFixture(t)
	f.pwc.EXPECT().DeleteByProfileName(context.Background(), "p1", "t1").Return(nil)
	f.repo.EXPECT().Delete(context.Background(), "p1", "t1").Return(true, nil)

	require.NoError(t, f.uc.Delete(context.Background(), "p1", "t1"))
	require.Equal(t, []string{"profiles/p1"}, f.store.deleted)
}
