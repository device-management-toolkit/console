package sqldb

import "github.com/device-management-toolkit/console/internal/repoerrors"

// ForeignKeyViolationError is an alias so both repository packages raise the
// same type and the controllers' errors.As checks work for either backend.
type ForeignKeyViolationError = repoerrors.ForeignKeyViolationError
