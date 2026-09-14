package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func serve(t *testing.T, headerValue string) (recorder *httptest.ResponseRecorder, seen string) {
	t.Helper()

	engine := gin.New()
	engine.Use(middleware.ResolveTenant(logger.New("error")))
	engine.GET("/", func(c *gin.Context) {
		seen = middleware.TenantID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	if headerValue != "" {
		req.Header.Set(middleware.TenantHeaderName, headerValue)
	}

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	return recorder, seen
}

func TestTenantAllowsValidHeader(t *testing.T) {
	t.Parallel()

	recorder, seen := serve(t, "tenant-a")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "tenant-a", seen)
}

func TestTenantWithoutHeaderYieldsEmptyTenant(t *testing.T) {
	t.Parallel()

	recorder, seen := serve(t, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, seen)
}

func TestTenantRejectsInvalidTenantIDHeader(t *testing.T) {
	t.Parallel()

	invalidTenantIDs := []struct {
		name  string
		value string
	}{
		{name: "embedded space", value: "tenant a"},
		{name: "leading hyphen", value: "-tenant-a"},
		{name: "leading underscore", value: "_tenant-a"},
		{name: "punctuation", value: "tenant.a"},
		{name: "over maximum length", value: strings.Repeat("a", middleware.MaxTenantIDLength+1)},
	}

	for _, test := range invalidTenantIDs {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder, seen := serve(t, test.value)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "x-tenant-id must match")
			require.Empty(t, seen)
		})
	}
}
