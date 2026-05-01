package sqldb

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

// ForeignKeyViolationError lives in sqldb (not internal/repoerrors) because
// foreign key constraints are a relational-only concept. Backends without
// referential integrity (e.g. MongoDB) do not produce this error, so it has
// no place in the cross-backend error vocabulary.
type ForeignKeyViolationError struct {
	Console consoleerrors.InternalError
}

func (e ForeignKeyViolationError) Error() string {
	return e.Console.Error()
}

func (e ForeignKeyViolationError) Wrap(details string) error {
	e.Console.Message = "foreign key violation: " + details

	return e
}
