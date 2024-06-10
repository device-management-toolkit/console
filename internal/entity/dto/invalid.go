package dto

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/open-amt-cloud-toolkit/console/pkg/consoleerrors"
)

type NotValidError struct {
	Console consoleerrors.InternalError
}

func (e NotValidError) Error() string {
	return e.Console.Error()
}

func (e NotValidError) Wrap(function, call string, err error) error {
	_ = e.Console.Wrap(function, call, err)
	e.Console.Message = "Invalid input: " + err.Error()

	return e
}

func CheckVersion(version string) error {
	if validator.New().Var(version, "required") != nil {
		return errors.New("version is required")
	}

	return nil
}
