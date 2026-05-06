// Package repoerrors holds the semantic error types every repository
// implementation produces. Backend packages (sqldb, future NoSQL backends)
// construct these so use cases can match on them with errors.As without
// coupling to a specific backend.
package repoerrors

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

type DatabaseError struct {
	Console consoleerrors.InternalError
}

func (e DatabaseError) Error() string {
	return e.Console.Error()
}

// Wrap records call/function/err on e.Console (mutated in place via pointer
// receiver on InternalError.Wrap) and returns e by value. The discarded return
// is intentional: only the side effect on e.Console is needed.
func (e DatabaseError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "database error"

	return e
}
