package repoerrors

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

// ForeignKeyViolationError reports a delete or insert that would break a
// relationship between records. Postgres and SQLite raise it from the foreign
// key constraint itself; MongoDB has no constraints, so its repositories check
// the referencing collection first and raise the same error — the way RPS did
// it (src/data/postgres/tables/wirelessProfiles.ts queries
// profiles_wirelessconfigs before deleting). Controllers map it to 400.
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
