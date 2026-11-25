package middleware

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

// brotliCompLevel matches nginx's brotli_comp_level in brotli.conf: the
// practical sweet spot for on-the-fly compression (level 11 is intended for
// static precompression, not per-request use).
const brotliCompLevel = 5

// compressibleExtensions lists static UI asset suffixes worth compressing.
// Binary/already-compressed assets (images, fonts, wasm) are left alone.
var compressibleExtensions = []string{
	".js", ".css", ".html", ".json", ".svg", ".map",
}

// Brotli compresses responses for compressible static assets when the client
// advertises Brotli support. The embedded UI is served directly from this Go
// binary (no reverse proxy in front of it), so without this middleware
// main.js/styles.css and friends go over the wire uncompressed.
func Brotli() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isCompressiblePath(c.Request.URL.Path) {
			c.Next()

			return
		}

		// Every compressible path varies on Accept-Encoding, including the
		// identity response, otherwise a cache could hand an uncompressed
		// copy to a Brotli client or vice versa. Append so that fields set by
		// earlier middleware (CORS adds Vary: Origin) survive.
		addVary(c.Writer.Header(), "Accept-Encoding")

		// Range offsets must refer to the bytes of the selected representation.
		// http.ServeContent slices the uncompressed file, so compressing that
		// slice would leave Content-Range pointing into the wrong byte stream.
		if c.GetHeader("Range") != "" || !acceptsBrotli(c.GetHeader("Accept-Encoding")) {
			c.Next()

			return
		}

		brWriter := brotli.NewWriterLevel(c.Writer, brotliCompLevel)
		defer brWriter.Close()

		c.Header("Content-Encoding", "br")
		c.Writer = &brotliResponseWriter{ResponseWriter: c.Writer, writer: brWriter}

		c.Next()
	}
}

func isCompressiblePath(path string) bool {
	for _, ext := range compressibleExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	return false
}

// addVary appends value to the Vary header unless it is already listed.
func addVary(h http.Header, value string) {
	for _, existing := range h.Values("Vary") {
		for _, field := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(field), value) {
				return
			}
		}
	}

	h.Add("Vary", value)
}

// acceptsBrotli parses an Accept-Encoding header (RFC 9110 §12.5.3) and
// reports whether brotli may be used. An explicit "br" entry wins over a
// wildcard, and a quality of 0 (e.g. "br;q=0") forbids the encoding.
func acceptsBrotli(acceptEncoding string) bool {
	wildcardAllowed := false

	for _, entry := range strings.Split(acceptEncoding, ",") {
		coding, params, _ := strings.Cut(entry, ";")
		coding = strings.ToLower(strings.TrimSpace(coding))

		switch coding {
		case "br":
			return qualityAllows(params)
		case "*":
			wildcardAllowed = qualityAllows(params)
		}
	}

	return wildcardAllowed
}

// qualityAllows returns false only when the parameters carry an explicit q=0
// weight; absent or malformed weights are treated as acceptable.
func qualityAllows(params string) bool {
	for _, param := range strings.Split(params, ";") {
		name, value, found := strings.Cut(param, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(name), "q") {
			continue
		}

		q, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return true
		}

		return q > 0
	}

	return true
}

// brotliResponseWriter wraps gin.ResponseWriter so writes go through brotli
// instead of straight to the client.
type brotliResponseWriter struct {
	gin.ResponseWriter
	writer io.Writer
}

// Write deletes Content-Length before the first write flushes headers: the
// handler (e.g. http.ServeContent) sets it from the uncompressed file size,
// which no longer matches the brotli-compressed body.
func (w *brotliResponseWriter) Write(data []byte) (int, error) {
	w.deleteStaleContentLength()

	return w.writer.Write(data)
}

func (w *brotliResponseWriter) WriteString(s string) (int, error) {
	w.deleteStaleContentLength()

	return io.WriteString(w.writer, s)
}

func (w *brotliResponseWriter) deleteStaleContentLength() {
	if !w.Written() {
		w.Header().Del("Content-Length")
	}
}

func (w *brotliResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

var _ http.ResponseWriter = (*brotliResponseWriter)(nil)
