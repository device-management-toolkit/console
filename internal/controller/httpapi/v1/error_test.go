package v1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func runErrorResponse(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	ErrorResponse(c, err)

	return w
}

func TestErrorResponse_CIRADisabled(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, profiles.ErrCIRADisabled)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestErrorResponse_CertDomainSuffix(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, domains.ErrCertDomainSuffix.Wrap("Insert", "CheckCertDomainSuffix", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "FQDN not associated with provisioning certificate")
}

func TestErrorResponse_CIRADeviceNotConnected(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, wsmanAPI.ErrCIRADeviceNotConnected)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestErrorResponse_InvalidProvisioningCertificate(t *testing.T) {
	t.Parallel()

	err := domains.ErrCertFormat.Wrap("test", "base64.StdEncoding.DecodeString", errors.New("illegal base64 data"))
	w := runErrorResponse(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"invalid provisioning certificate","message":"invalid provisioning certificate"}`, w.Body.String())
}

func TestErrorResponse_JSONBindingError(t *testing.T) {
	t.Parallel()

	for _, requestBody := range []string{
		`"fuzzstring"`,
		`{"action":false}`,
		`{"action":"fuzzstring"}`,
		`{"bootPath":"\OemPba.efi"}`,
	} {
		var powerAction dto.PowerAction

		err := json.Unmarshal([]byte(requestBody), &powerAction)
		w := runErrorResponse(t, wrapRequestBindingError(err))

		assert.Equal(t, http.StatusBadRequest, w.Code)
	}
}

func TestErrorResponse_WrappedJSONBindingError(t *testing.T) {
	t.Parallel()

	var device dto.Device

	err := json.Unmarshal([]byte(`{"tags":"test"}`), &device)
	err = dto.NotValidError{Console: consoleerrors.CreateConsoleError("ProfileAPI")}.Wrap("insert", "json.Unmarshal", err)
	w := runErrorResponse(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Invalid input: json: cannot unmarshal string into Go struct field Device.tags of type []string","message":"Invalid input: json: cannot unmarshal string into Go struct field Device.tags of type []string"}`, w.Body.String())
}

func TestErrorResponse_InvalidTimestamp(t *testing.T) {
	t.Parallel()

	var alarm dto.AlarmClockOccurrenceInput

	err := json.Unmarshal([]byte(`{"StartTime":"fuzzstring"}`), &alarm)
	w := runErrorResponse(t, wrapRequestBindingError(err))

	var parseErr *time.ParseError
	assert.ErrorAs(t, err, &parseErr)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestErrorResponse_BackendEOF(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, io.EOF)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestErrorResponse_RequestEOF(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, wrapRequestBindingError(io.EOF))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"request body is empty","message":"request body is empty"}`, w.Body.String())
}

func TestErrorResponse_RequestTruncatedBody(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, wrapRequestBindingError(io.ErrUnexpectedEOF))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"request body is truncated","message":"request body is truncated"}`, w.Body.String())
}

// A binding failure is a client error regardless of the underlying error type.
// gin returns a bare errors.New for a nil body, and encoding/json does the same
// for unknown fields; neither is a *json.SyntaxError.
func TestErrorResponse_RequestBindingErrorOfUnrecognizedType(t *testing.T) {
	t.Parallel()

	for _, inner := range []error{
		errors.New("invalid request"),
		errors.New(`json: unknown field "foo"`),
	} {
		w := runErrorResponse(t, wrapRequestBindingError(inner))

		var got response

		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, inner.Error(), got.Error)
		assert.Equal(t, inner.Error(), got.Message)
	}
}

// The same unrecognized shapes coming from the backend must stay 5xx.
func TestErrorResponse_BackendErrorOfUnrecognizedType(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, errors.New("wsman transport failure"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleSentinelErrors_CIRADeviceNotConnected(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	handled := handleSentinelErrors(c, wsmanAPI.ErrCIRADeviceNotConnected)
	assert.True(t, handled)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleSentinelErrors_UnknownError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	handled := handleSentinelErrors(c, errors.New("some unknown error"))
	assert.False(t, handled)
}
