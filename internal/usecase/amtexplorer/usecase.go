package amtexplorer

import (
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var ErrDatabase = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("DevicesUseCase")}

// UseCase -.
type UseCase struct {
	repo             Repository
	device           WSMAN
	log              logger.Interface
	safeRequirements security.Cryptor
}

var ErrAMT = AMTError{Console: consoleerrors.CreateConsoleError("DevicesUseCase")}

// New -.
func New(r Repository, d WSMAN, log logger.Interface, safeRequirements security.Cryptor) *UseCase {
	return &UseCase{
		repo:             r,
		device:           d,
		log:              log,
		safeRequirements: safeRequirements,
	}
}
