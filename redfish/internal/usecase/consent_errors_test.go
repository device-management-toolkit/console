package usecase

import (
	"strings"
	"testing"
)

func TestConsentFailedErrorMessage_KnownReturnValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        *ConsentFailedError
		wantSubstr string
	}{
		{
			name:       "request return value 2",
			err:        &ConsentFailedError{Operation: consentOperationRequest, ReturnValue: consentReturnValueInvalidState},
			wantSubstr: "cannot request consent in the current opt-in state",
		},
		{
			name:       "submit return value 2",
			err:        &ConsentFailedError{Operation: consentOperationSubmit, ReturnValue: consentReturnValueInvalidState},
			wantSubstr: "cannot submit consent code in the current opt-in state",
		},
		{
			name:       "cancel return value 2",
			err:        &ConsentFailedError{Operation: consentOperationCancel, ReturnValue: consentReturnValueInvalidState},
			wantSubstr: "cannot cancel consent in the current opt-in state",
		},
		{
			name:       "unknown operation return value 2",
			err:        &ConsentFailedError{Operation: "OtherConsentOp", ReturnValue: consentReturnValueInvalidState},
			wantSubstr: "operation is not allowed in the current AMT opt-in state",
		},
		{
			name:       "request return value 3",
			err:        &ConsentFailedError{Operation: consentOperationRequest, ReturnValue: consentReturnValueOperationNotReady},
			wantSubstr: "cannot request consent because AMT is not ready",
		},
		{
			name:       "submit return value 3",
			err:        &ConsentFailedError{Operation: consentOperationSubmit, ReturnValue: consentReturnValueOperationNotReady},
			wantSubstr: "cannot submit consent code because AMT is not ready",
		},
		{
			name:       "cancel return value 3",
			err:        &ConsentFailedError{Operation: consentOperationCancel, ReturnValue: consentReturnValueOperationNotReady},
			wantSubstr: "cannot cancel consent because AMT is not ready",
		},
		{
			name:       "unknown operation return value 3",
			err:        &ConsentFailedError{Operation: "OtherConsentOp", ReturnValue: consentReturnValueOperationNotReady},
			wantSubstr: "operation cannot proceed because AMT is not ready",
		},
		{
			name:       "submit return value 2066",
			err:        &ConsentFailedError{Operation: consentOperationSubmit, ReturnValue: consentReturnValueConsentCodeInvalid},
			wantSubstr: "consent code was rejected by AMT",
		},
		{
			name:       "unknown operation return value 2066",
			err:        &ConsentFailedError{Operation: "OtherConsentOp", ReturnValue: consentReturnValueConsentCodeInvalid},
			wantSubstr: "operation failed due to AMT consent validation error",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.err.Error()
			if !strings.Contains(got, tt.wantSubstr) {
				t.Fatalf("ConsentFailedError.Error() = %q, want substring %q", got, tt.wantSubstr)
			}
		})
	}
}

func TestConsentFailedErrorMessage_FallbackFormat(t *testing.T) {
	t.Parallel()

	err := &ConsentFailedError{Operation: consentOperationRequest, ReturnValue: 5}
	got := err.Error()
	want := "RequestKVMConsent failed with ReturnValue=5"

	if got != want {
		t.Fatalf("ConsentFailedError.Error() = %q, want %q", got, want)
	}
}

func TestConsentFailedErrorMessage_SolOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        *ConsentFailedError
		wantSubstr string
	}{
		{
			name:       "sol request return value 2",
			err:        &ConsentFailedError{Operation: consentOperationSolRequest, ReturnValue: consentReturnValueInvalidState},
			wantSubstr: "cannot request consent in the current opt-in state",
		},
		{
			name:       "sol submit return value 2",
			err:        &ConsentFailedError{Operation: consentOperationSolSubmit, ReturnValue: consentReturnValueInvalidState},
			wantSubstr: "cannot submit consent code in the current opt-in state",
		},
		{
			name:       "sol cancel return value 2",
			err:        &ConsentFailedError{Operation: consentOperationSolCancel, ReturnValue: consentReturnValueInvalidState},
			wantSubstr: "cannot cancel consent in the current opt-in state",
		},
		{
			name:       "sol request return value 3",
			err:        &ConsentFailedError{Operation: consentOperationSolRequest, ReturnValue: consentReturnValueOperationNotReady},
			wantSubstr: "cannot request consent because AMT is not ready",
		},
		{
			name:       "sol submit return value 3",
			err:        &ConsentFailedError{Operation: consentOperationSolSubmit, ReturnValue: consentReturnValueOperationNotReady},
			wantSubstr: "cannot submit consent code because AMT is not ready",
		},
		{
			name:       "sol cancel return value 3",
			err:        &ConsentFailedError{Operation: consentOperationSolCancel, ReturnValue: consentReturnValueOperationNotReady},
			wantSubstr: "cannot cancel consent because AMT is not ready",
		},
		{
			name:       "sol submit return value 2066",
			err:        &ConsentFailedError{Operation: consentOperationSolSubmit, ReturnValue: consentReturnValueConsentCodeInvalid},
			wantSubstr: "consent code was rejected by AMT",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.err.Error()
			if !strings.Contains(got, tt.wantSubstr) {
				t.Fatalf("ConsentFailedError.Error() = %q, want substring %q", got, tt.wantSubstr)
			}
		})
	}
}

func TestConsentFailedErrorMessage_SolFallbackFormat(t *testing.T) {
	t.Parallel()

	err := &ConsentFailedError{Operation: consentOperationSolRequest, ReturnValue: 5}
	got := err.Error()
	want := "RequestSolConsent failed with ReturnValue=5"

	if got != want {
		t.Fatalf("ConsentFailedError.Error() = %q, want %q", got, want)
	}
}
