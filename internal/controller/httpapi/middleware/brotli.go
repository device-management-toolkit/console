package middleware

import (
	"io"
	"net/http"
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
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "br") {
			c.Next()

			return
		}

		if !isCompressiblePath(c.Request.URL.Path) {
			c.Next()

			return
		}

		brWriter := brotli.NewWriterLevel(c.Writer, brotliCompLevel)
		defer brWriter.Close()

		c.Header("Content-Encoding", "br")
		c.Header("Vary", "Accept-Encoding")
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
