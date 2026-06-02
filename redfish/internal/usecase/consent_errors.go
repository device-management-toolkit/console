package usecase

import "fmt"

const (
	consentOperationRequest = "RequestKVMConsent"
	consentOperationSubmit  = "SubmitKVMConsentCode"
	consentOperationCancel  = "CancelKVMConsent"

	consentReturnValueInvalidState       = 2
	consentReturnValueOperationNotReady  = 3
	consentReturnValueConsentCodeInvalid = 2066
)

// ConsentFailedError indicates that AMT processed a consent operation but returned a non-zero ReturnValue.
type ConsentFailedError struct {
	Operation   string
	ReturnValue int
}

func (e *ConsentFailedError) Error() string {
	if msg, ok := consentReturnValueMessage(e.Operation, e.ReturnValue); ok {
		return fmt.Sprintf("%s failed with ReturnValue=%d: %s", e.Operation, e.ReturnValue, msg)
	}

	return fmt.Sprintf("%s failed with ReturnValue=%d", e.Operation, e.ReturnValue)
}

func consentReturnValueMessage(operation string, returnValue int) (string, bool) {
	switch returnValue {
	case consentReturnValueInvalidState:
		switch operation {
		case consentOperationRequest:
			return "cannot request consent in the current opt-in state", true
		case consentOperationSubmit:
			return "cannot submit consent code in the current opt-in state", true
		case consentOperationCancel:
			return "cannot cancel consent in the current opt-in state", true
		default:
			return "operation is not allowed in the current AMT opt-in state", true
		}
	case consentReturnValueOperationNotReady:
		switch operation {
		case consentOperationRequest:
			return "cannot request consent because AMT is not ready for this operation in the current configuration/state", true
		case consentOperationSubmit:
			return "cannot submit consent code because AMT is not ready for this operation in the current configuration/state", true
		case consentOperationCancel:
			return "cannot cancel consent because AMT is not ready for this operation in the current configuration/state", true
		default:
			return "operation cannot proceed because AMT is not ready in the current configuration/state", true
		}
	case consentReturnValueConsentCodeInvalid:
		switch operation {
		case consentOperationSubmit:
			return "consent code was rejected by AMT", true
		default:
			return "operation failed due to AMT consent validation error", true
		}
	default:
		return "", false
	}
}
