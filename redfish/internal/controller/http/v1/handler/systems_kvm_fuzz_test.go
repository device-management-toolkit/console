package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// kvmConsentErrorFor maps a fuzz selector to an injected repository error.
// Each branch exercises a distinct arm of the consent error-handling switch
// in the KVM consent action handlers (success, not-found, consent failure,
// ACM-not-required, AMT 400, and generic internal failure).
func kvmConsentErrorFor(selector uint8) error {
	switch selector % 6 {
	case 1:
		return usecase.ErrSystemNotFound
	case 2:
		return &usecase.ConsentFailedError{Operation: "RequestKVMConsent", ReturnValue: 2}
	case 3:
		return usecase.ErrKVMConsentNotRequiredInACM
	case 4:
		return errors.New("AMT call failed: 400 Bad Request")
	case 5:
		return errors.New("unexpected AMT failure")
	default:
		return nil
	}
}

// kvmUpdateErrorFor maps a fuzz selector to an injected GraphicalConsole update error.
func kvmUpdateErrorFor(selector uint8) error {
	switch selector % 3 {
	case 1:
		return usecase.ErrSystemNotFound
	case 2:
		return errors.New("graphical console update failed")
	default:
		return nil
	}
}

// disableRouterRedirects turns off gin's path-normalization redirects so fuzzed
// request paths (e.g. ones that decode to empty or doubled segments) reach the
// handler instead of short-circuiting with a router-level 3xx redirect. This keeps
// the fuzz focused on handler logic rather than gin's routing layer.
func disableRouterRedirects(router *gin.Engine) *gin.Engine {
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	return router
}

// newKVMConsentFuzzRouter builds a router backed by a repository seeded with the
// canonical test system. When injectedErr is non-nil it is returned by the
// consent repository methods for that system, driving the handler error paths.
func newKVMConsentFuzzRouter(injectedErr error) *gin.Engine {
	repo := NewTestSystemsComputerSystemRepository()
	repo.AddSystem(testSystemID, &redfishv1.ComputerSystem{ID: testSystemID, Name: "Test System"})

	if injectedErr != nil {
		repo.errorOnGetByID[testSystemID] = injectedErr
	}

	server := setupSystemActionsTestServer(repo)

	return disableRouterRedirects(setupSystemActionsTestRouter(server))
}

// assertValidKVMActionStatus fails the fuzz iteration if the handler produced a
// status code outside the contract for KVM action/PATCH endpoints. Any panic in
// the handler chain surfaces as a test failure automatically.
func assertValidKVMActionStatus(t *testing.T, w *httptest.ResponseRecorder, systemID, body string) {
	t.Helper()

	switch w.Code {
	case http.StatusOK, http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError:
	default:
		t.Fatalf("unexpected status %d for systemID %q body %q", w.Code, systemID, body)
	}
}

// FuzzRequestKVMConsentHandler fuzzes the RequestKVMConsent OEM action handler with
// arbitrary system IDs, request bodies, and injected repository errors.
// Verifies: no panics and only contract-defined status codes are returned.
func FuzzRequestKVMConsentHandler(f *testing.F) {
	seedInputs := []struct {
		systemID string
		body     string
		errSel   uint8
	}{
		{testSystemID, `{}`, 0},
		{testSystemID, ``, 0},
		{testSystemID, `{}`, 1},
		{testSystemID, `{}`, 2},
		{testSystemID, `{}`, 3},
		{testSystemID, `{}`, 4},
		{testSystemID, `{}`, 5},
		{testSystemID, `not-json`, 0},
		{testSystemID, `{"unexpected":true}`, 0},
		{testUUIDNotFound, `{}`, 0},
		{"not-a-uuid", `{}`, 0},
		{"", `{}`, 0},
		{"/", `{}`, 0},
		{"../etc/passwd", `{}`, 0},
		{"用戶🙂", `{}`, 0},
	}

	for _, in := range seedInputs {
		f.Add(in.systemID, in.body, in.errSel)
	}

	f.Fuzz(func(t *testing.T, systemID, body string, errSel uint8) {
		router := newKVMConsentFuzzRouter(kvmConsentErrorFor(errSel))

		target := "/redfish/v1/Systems/" + url.PathEscape(systemID) + "/Actions/Oem/IntelComputerSystem.RequestKVMConsent"
		req := httptest.NewRequest(http.MethodPost, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assertValidKVMActionStatus(t, w, systemID, body)
	})
}

// FuzzCancelKVMConsentHandler fuzzes the CancelKVMConsent OEM action handler with
// arbitrary system IDs, request bodies, and injected repository errors.
// Verifies: no panics and only contract-defined status codes are returned.
func FuzzCancelKVMConsentHandler(f *testing.F) {
	seedInputs := []struct {
		systemID string
		body     string
		errSel   uint8
	}{
		{testSystemID, `{}`, 0},
		{testSystemID, ``, 0},
		{testSystemID, `{}`, 1},
		{testSystemID, `{}`, 2},
		{testSystemID, `{}`, 4},
		{testSystemID, `{}`, 5},
		{testSystemID, `not-json`, 0},
		{testUUIDNotFound, `{}`, 0},
		{"not-a-uuid", `{}`, 0},
		{"", `{}`, 0},
		{"/", `{}`, 0},
		{"用戶🙂", `{}`, 0},
	}

	for _, in := range seedInputs {
		f.Add(in.systemID, in.body, in.errSel)
	}

	f.Fuzz(func(t *testing.T, systemID, body string, errSel uint8) {
		router := newKVMConsentFuzzRouter(kvmConsentErrorFor(errSel))

		target := "/redfish/v1/Systems/" + url.PathEscape(systemID) + "/Actions/Oem/IntelComputerSystem.CancelKVMConsent"
		req := httptest.NewRequest(http.MethodPost, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assertValidKVMActionStatus(t, w, systemID, body)
	})
}

// FuzzSubmitKVMConsentCodeHandler fuzzes the SubmitKVMConsentCode OEM action handler
// with arbitrary system IDs, consent codes, and injected repository errors. This
// exercises the six-digit consent-code validation as well as the consent error paths.
// Verifies: no panics and only contract-defined status codes are returned.
func FuzzSubmitKVMConsentCodeHandler(f *testing.F) {
	seedInputs := []struct {
		systemID string
		code     string
		errSel   uint8
	}{
		{testSystemID, "123456", 0},
		{testSystemID, "000000", 0},
		{testSystemID, "123456", 1},
		{testSystemID, "123456", 2},
		{testSystemID, "123456", 4},
		{testSystemID, "123456", 5},
		{testSystemID, "", 0},
		{testSystemID, "12345", 0},
		{testSystemID, "1234567", 0},
		{testSystemID, "abcdef", 0},
		{testSystemID, "12 34 56", 0},
		{testSystemID, " 123456 ", 0},
		{testSystemID, "１２３４５６", 0},
		{testUUIDNotFound, "123456", 0},
		{"not-a-uuid", "123456", 0},
		{"", "123456", 0},
		{"/", "123456", 0},
	}

	for _, in := range seedInputs {
		f.Add(in.systemID, in.code, in.errSel)
	}

	f.Fuzz(func(t *testing.T, systemID, code string, errSel uint8) {
		router := newKVMConsentFuzzRouter(kvmConsentErrorFor(errSel))
		body := createSubmitKVMConsentCodeRequest(code)

		target := "/redfish/v1/Systems/" + url.PathEscape(systemID) + "/Actions/Oem/IntelComputerSystem.SubmitKVMConsentCode"
		req := httptest.NewRequest(http.MethodPost, target, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assertValidKVMActionStatus(t, w, systemID, code)
	})
}

// FuzzSubmitKVMConsentCodeRawBody fuzzes the SubmitKVMConsentCode handler with raw,
// possibly-malformed request bodies to harden the JSON binding path against panics.
// Verifies: no panics and only contract-defined status codes are returned.
func FuzzSubmitKVMConsentCodeRawBody(f *testing.F) {
	seeds := []string{
		`{"ConsentCode":"123456"}`,
		`{"ConsentCode":""}`,
		`{"ConsentCode":123456}`,
		`{"ConsentCode":null}`,
		`{}`,
		``,
		`not-json`,
		`{"ConsentCode":"123456"`,
		`[1,2,3]`,
		`{"ConsentCode":"١٢٣٤٥٦"}`,
		`{"ConsentCode":"\u0000\u0000\u0000\u0000\u0000\u0000"}`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, rawBody string) {
		router := newKVMConsentFuzzRouter(nil)

		target := "/redfish/v1/Systems/" + testSystemID + "/Actions/Oem/IntelComputerSystem.SubmitKVMConsentCode"
		req := httptest.NewRequest(http.MethodPost, target, bytes.NewBufferString(rawBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assertValidKVMActionStatus(t, w, testSystemID, rawBody)
	})
}

// FuzzSubmitKVMConsentCodeJSON fuzzes JSON unmarshalling of the SubmitKVMConsentCode
// request body type. Verifies: no panics, deterministic parse results, and that a
// successful parse yields a stable ConsentCode value.
func FuzzSubmitKVMConsentCodeJSON(f *testing.F) {
	seeds := []string{
		`{"ConsentCode":"123456"}`,
		`{"ConsentCode":""}`,
		`{"ConsentCode":null}`,
		`{"ConsentCode":123456}`,
		`{}`,
		`{"ConsentCode":"123456","Extra":true}`,
		`not-json`,
		`{"ConsentCode":"١٢٣٤٥٦"}`,
		`{"ConsentCode":"\u0000\u0000\u0000\u0000\u0000\u0000"}`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		var (
			first  generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitKVMConsentCodeJSONRequestBody
			second generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitKVMConsentCodeJSONRequestBody
		)

		firstErr := json.Unmarshal([]byte(payload), &first)
		secondErr := json.Unmarshal([]byte(payload), &second)

		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("non-deterministic error for payload %q: first=%v second=%v", payload, firstErr, secondErr)
		}

		if firstErr != nil {
			return
		}

		if first.ConsentCode != second.ConsentCode {
			t.Fatalf("non-deterministic unmarshal for payload %q: %q vs %q", payload, first.ConsentCode, second.ConsentCode)
		}
	})
}

// FuzzGenerateRedirectionTokenHandler fuzzes the GenerateRedirectionToken OEM action
// handler (used to authorize KVM/SOL redirection) with arbitrary system IDs and bodies.
// Verifies: no panics and only contract-defined status codes are returned.
func FuzzGenerateRedirectionTokenHandler(f *testing.F) {
	seedInputs := []struct {
		systemID string
		body     string
	}{
		{testSystemID, `{}`},
		{testSystemID, ``},
		{testSystemID, `not-json`},
		{testUUIDNotFound, `{}`},
		{"not-a-uuid", `{}`},
		{"", `{}`},
		{"用戶🙂", `{}`},
	}

	for _, in := range seedInputs {
		f.Add(in.systemID, in.body)
	}

	f.Fuzz(func(t *testing.T, systemID, body string) {
		configureTestJWT(t)

		repo := NewTestSystemsComputerSystemRepository()
		repo.AddSystem(testSystemID, &redfishv1.ComputerSystem{ID: testSystemID, Name: "Test System"})
		server := setupSystemActionsTestServer(repo)
		router := disableRouterRedirects(setupSystemActionsTestRouter(server))

		target := "/redfish/v1/Systems/" + url.PathEscape(systemID) + "/Actions/Oem/IntelComputerSystem.GenerateRedirectionToken"
		req := httptest.NewRequest(http.MethodPost, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assertValidKVMActionStatus(t, w, systemID, body)
	})
}

// FuzzGraphicalConsoleServiceEnabledHandler fuzzes the PATCH ComputerSystem handler that
// enables or disables the KVM (GraphicalConsole) service, with arbitrary system IDs,
// bodies, and injected update errors.
// Verifies: no panics and only contract-defined status codes are returned.
func FuzzGraphicalConsoleServiceEnabledHandler(f *testing.F) {
	seedInputs := []struct {
		systemID string
		body     string
		errSel   uint8
	}{
		{testSystemID, `{"GraphicalConsole":{"ServiceEnabled":true}}`, 0},
		{testSystemID, `{"GraphicalConsole":{"ServiceEnabled":false}}`, 0},
		{testSystemID, `{"GraphicalConsole":{"ServiceEnabled":true}}`, 1},
		{testSystemID, `{"GraphicalConsole":{"ServiceEnabled":true}}`, 2},
		{testSystemID, `{"GraphicalConsole":{}}`, 0},
		{testSystemID, `{"GraphicalConsole":null}`, 0},
		{testSystemID, `{}`, 0},
		{testSystemID, `not-json`, 0},
		{testSystemID, `{"GraphicalConsole":{"ServiceEnabled":"yes"}}`, 0},
		{testUUIDNotFound, `{"GraphicalConsole":{"ServiceEnabled":true}}`, 0},
		{"not-a-uuid", `{"GraphicalConsole":{"ServiceEnabled":true}}`, 0},
		{"", `{"GraphicalConsole":{"ServiceEnabled":true}}`, 0},
		{"/", `{"GraphicalConsole":{"ServiceEnabled":true}}`, 0},
	}

	for _, in := range seedInputs {
		f.Add(in.systemID, in.body, in.errSel)
	}

	f.Fuzz(func(t *testing.T, systemID, body string, errSel uint8) {
		repo := NewTestSystemsComputerSystemRepository()
		repo.AddSystem(testSystemID, &redfishv1.ComputerSystem{ID: testSystemID, Name: "Test System"})

		if injectedErr := kvmUpdateErrorFor(errSel); injectedErr != nil {
			repo.errorOnUpdateKVM[testSystemID] = injectedErr
		}

		server := setupSystemActionsTestServer(repo)
		router := disableRouterRedirects(setupPatchSystemTestRouter(server))

		target := "/redfish/v1/Systems/" + url.PathEscape(systemID)
		req := httptest.NewRequest(http.MethodPatch, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assertValidKVMActionStatus(t, w, systemID, body)
	})
}
