package v1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
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

func TestErrorResponse_CIRADeviceNotConnected(t *testing.T) {
	t.Parallel()

	w := runErrorResponse(t, wsmanAPI.ErrCIRADeviceNotConnected)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
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
