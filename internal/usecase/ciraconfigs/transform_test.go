package ciraconfigs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestDTOEntityRoundTrip_GenerateRandomPassword(t *testing.T) {
	t.Parallel()

	uc := &UseCase{log: logger.New("error"), safeRequirements: mocks.MockCrypto{}}

	for _, value := range []bool{true, false} {
		value := value

		t.Run("", func(t *testing.T) {
			t.Parallel()

			input := &dto.CIRAConfig{
				ConfigName:             "cfg",
				TenantID:               "tenant",
				GenerateRandomPassword: value,
			}

			ent, err := uc.dtoToEntity(input)
			require.NoError(t, err)
			assert.Equal(t, value, ent.GenerateRandomPassword)

			back := uc.entityToDTO(ent)
			assert.Equal(t, value, back.GenerateRandomPassword)
		})
	}
}
