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

type brotliRequest struct {
	path           string
	acceptEncoding string
	rangeHeader    string
	// priorVary simulates an upstream middleware (e.g. CORS) that already
	// populated Vary before Brotli runs.
	priorVary string
}

func serveBrotli(t *testing.T, path, acceptEncoding string) *httptest.ResponseRecorder {
	t.Helper()

	return serveBrotliRequest(t, brotliRequest{path: path, acceptEncoding: acceptEncoding})
}

func serveBrotliRequest(t *testing.T, r brotliRequest) *httptest.ResponseRecorder {
	t.Helper()

	engine := gin.New()

	if r.priorVary != "" {
		engine.Use(func(c *gin.Context) {
			c.Writer.Header().Add("Vary", r.priorVary)
			c.Next()
		})
	}

	engine.Use(middleware.Brotli())
	engine.GET(r.path, func(c *gin.Context) {
		c.String(http.StatusOK, bodyText)
	})

	req := httptest.NewRequest(http.MethodGet, r.path, http.NoBody)
	if r.acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", r.acceptEncoding)
	}

	if r.rangeHeader != "" {
		req.Header.Set("Range", r.rangeHeader)
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
	require.Equal(t, []string{"Accept-Encoding"}, recorder.Header().Values("Vary"))
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

func TestBrotliIdentityResponseStillVariesOnAcceptEncoding(t *testing.T) {
	t.Parallel()

	recorder := serveBrotli(t, "/main.js", "gzip")

	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	require.Equal(t, []string{"Accept-Encoding"}, recorder.Header().Values("Vary"))
}

func TestBrotliAcceptEncodingNegotiation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		acceptEncoding string
		wantBrotli     bool
	}{
		{name: "explicit zero quality forbids br", acceptEncoding: "gzip, br;q=0", wantBrotli: false},
		{name: "zero quality with decimals forbids br", acceptEncoding: "br;q=0.000", wantBrotli: false},
		{name: "positive quality allows br", acceptEncoding: "gzip;q=1.0, br;q=0.5", wantBrotli: true},
		{name: "case insensitive coding", acceptEncoding: "BR", wantBrotli: true},
		{name: "wildcard allows br", acceptEncoding: "*", wantBrotli: true},
		{name: "wildcard with zero quality forbids br", acceptEncoding: "gzip, *;q=0", wantBrotli: false},
		{name: "explicit br wins over forbidden wildcard", acceptEncoding: "br, *;q=0", wantBrotli: true},
		{name: "explicit br zero wins over wildcard", acceptEncoding: "*, br;q=0", wantBrotli: false},
		{name: "substring is not a match", acceptEncoding: "brotli-ish", wantBrotli: false},
		{name: "empty header", acceptEncoding: "", wantBrotli: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recorder := serveBrotli(t, "/main.js", tc.acceptEncoding)

			require.Equal(t, http.StatusOK, recorder.Code)

			if tc.wantBrotli {
				require.Equal(t, "br", recorder.Header().Get("Content-Encoding"))

				decoded, err := io.ReadAll(brotli.NewReader(recorder.Body))
				require.NoError(t, err)
				require.Equal(t, bodyText, string(decoded))
			} else {
				require.Empty(t, recorder.Header().Get("Content-Encoding"))
				require.Equal(t, bodyText, recorder.Body.String())
			}
		})
	}
}

func TestBrotliBypassesRangeRequests(t *testing.T) {
	t.Parallel()

	recorder := serveBrotliRequest(t, brotliRequest{
		path:           "/main.js",
		acceptEncoding: "br",
		rangeHeader:    "bytes=0-99",
	})

	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	require.Equal(t, bodyText, recorder.Body.String())
	require.Equal(t, []string{"Accept-Encoding"}, recorder.Header().Values("Vary"))
}

func TestBrotliPreservesExistingVaryHeader(t *testing.T) {
	t.Parallel()

	recorder := serveBrotliRequest(t, brotliRequest{
		path:           "/main.js",
		acceptEncoding: "br",
		priorVary:      "Origin",
	})

	require.Equal(t, "br", recorder.Header().Get("Content-Encoding"))
	require.Equal(t, []string{"Origin", "Accept-Encoding"}, recorder.Header().Values("Vary"))
}

func TestBrotliDoesNotDuplicateVaryAcceptEncoding(t *testing.T) {
	t.Parallel()

	recorder := serveBrotliRequest(t, brotliRequest{
		path:           "/main.js",
		acceptEncoding: "br",
		priorVary:      "Origin, accept-encoding",
	})

	require.Equal(t, []string{"Origin, accept-encoding"}, recorder.Header().Values("Vary"))
}

func TestBrotliSkipsNonCompressiblePath(t *testing.T) {
	t.Parallel()

	recorder := serveBrotli(t, "/assets/logo.png", "br")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("Content-Encoding"))
	require.Empty(t, recorder.Header().Values("Vary"))
	require.Equal(t, bodyText, recorder.Body.String())
}
