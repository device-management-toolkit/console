package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
)

const bodyText = "console main.js payload"

func serveBrotli(t *testing.T, path, acceptEncoding string) *httptest.ResponseRecorder {
	t.Helper()

	engine := gin.New()
	engine.Use(middleware.Brotli())
	engine.GET(path, func(c *gin.Context) {
		c.String(http.StatusOK, bodyText)
	})

	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	return recorder
}

func TestBrotliCompressesCompressibleAssetWhenAccepted(t *testing.T) {
	t.Parallel()

	recorder := serveBrotli(t, "/main.js", "gzip, deflate, br")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "br", recorder.Header().Get("Content-Encoding"))
	require.Equal(t, "Accept-Encoding", recorder.Header().Get("Vary"))
	require.Empty(t, recorder.Header().Get("Content-Length"))

	reader := brotli.NewReader(recorder.Body)

	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, bodyText, string(decoded))
}

func TestBrotliSkipsWhenClientDoesNotAcceptIt(t *testing.T) {
	t.Parallel()

	recorder := serveBrotli(t, "/main.js", "gzip, deflate")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	require.Equal(t, bodyText, recorder.Body.String())
}

func TestBrotliSkipsNonCompressiblePath(t *testing.T) {
	t.Parallel()

	recorder := serveBrotli(t, "/assets/logo.png", "br")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	require.Equal(t, bodyText, recorder.Body.String())
}
