package consoleerrors

import (
	"fmt"
)

type InternalError struct {
	file          string
	Function      string
	Call          string
	Message       string
	InnerTrace    string
	OriginalError error
}

func (e InternalError) Error() string {
	errMsg := ""

	if e.OriginalError != nil {
		errMsg = e.OriginalError.Error()
	}

	return fmt.Sprintf("%s - %s - %s: %s", e.file, e.Function, e.Call, errMsg)
}

func (e InternalError) FriendlyMessage() string {
	return e.Message
}

func (e *InternalError) Wrap(call, function string, err error) error {
	e.Call = call
	e.Function = function

	e.OriginalError = err
	if err != nil {
		e.InnerTrace = "" // err.Error()
	}

	return e
}

func CreateConsoleError(file string) InternalError {
	message := ""

	return InternalError{
		file:    file,
		Message: message,
	}
}
