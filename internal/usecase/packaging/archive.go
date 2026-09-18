package packaging

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	binaryName      = "rpc"
	binaryNameWin   = "rpc.exe"
	configFileName  = "config.yaml"
	binaryFileMode  = 0o755
	maxArchiveBytes = 200 << 20 // 200 MiB cap to guard against decompression bombs
	httpTimeout     = 30 * time.Second
)

// httpClient is shared by the releases list and asset downloads.
var httpClient = &http.Client{Timeout: httpTimeout} //nolint:gochecknoglobals // shared so connections are reused

var (
	// ErrBinaryNotFound indicates the archive did not hold the expected rpc binary.
	ErrBinaryNotFound = errors.New("rpc binary not found in archive")
	// ErrEntryTooLarge indicates an archive entry exceeded the size cap.
	ErrEntryTooLarge = errors.New("archive entry exceeds size limit")
	// ErrDownloadAsset indicates a non-200 response downloading an asset.
	ErrDownloadAsset = errors.New("failed to download asset")
)

// readLimited reads up to maxArchiveBytes from r, guarding against decompression bombs.
func readLimited(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer

	n, err := io.CopyN(&buf, r, maxArchiveBytes+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read entry: %w", err)
	}

	if n > maxArchiveBytes {
		return nil, ErrEntryTooLarge
	}

	return buf.Bytes(), nil
}

// extractBinary returns the packaged name and bytes of an rpc-go asset: a bare
// .exe, or a .tar.gz holding one file named after the asset.
func extractBinary(data []byte, assetName string) (name string, content []byte, err error) {
	if strings.HasSuffix(assetName, ".exe") {
		return binaryNameWin, data, nil
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	hdr, err := tr.Next()
	if err != nil {
		return "", nil, fmt.Errorf("tar: %w", err)
	}

	if hdr.Typeflag != tar.TypeReg || hdr.Name != strings.TrimSuffix(assetName, ".tar.gz") {
		return "", nil, fmt.Errorf("%w: %s", ErrBinaryNotFound, hdr.Name)
	}

	content, err = readLimited(tr)
	if err != nil {
		return "", nil, err
	}

	return binaryName, content, nil
}

// zipEntry is one binary placed in the downloadable zip.
type zipEntry struct {
	name    string
	content []byte
}

// buildZip assembles the downloadable zip containing the binaries and config.yaml.
func buildZip(binaries []zipEntry, configYAML []byte) ([]byte, error) {
	var buf bytes.Buffer

	size := len(configYAML)
	for _, b := range binaries {
		size += len(b.content)
	}

	buf.Grow(size)

	zw := zip.NewWriter(&buf)

	for _, b := range binaries {
		binHeader := &zip.FileHeader{Name: b.name, Method: zip.Deflate}
		binHeader.SetMode(binaryFileMode)

		bw, err := zw.CreateHeader(binHeader)
		if err != nil {
			return nil, fmt.Errorf("zip create binary: %w", err)
		}

		if _, err := bw.Write(b.content); err != nil {
			return nil, fmt.Errorf("zip write binary: %w", err)
		}
	}

	cw, err := zw.Create(configFileName)
	if err != nil {
		return nil, fmt.Errorf("zip create config: %w", err)
	}

	if _, err := cw.Write(configYAML); err != nil {
		return nil, fmt.Errorf("zip write config: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("zip close: %w", err)
	}

	return buf.Bytes(), nil
}

// httpGet GETs url and returns the response body; a non-200 status wraps sentinel.
func httpGet(ctx context.Context, url string, sentinel error) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()

		return nil, fmt.Errorf("%w: %s", sentinel, resp.Status)
	}

	return resp.Body, nil
}

// downloadAsset fetches an asset's bytes over HTTP.
func downloadAsset(ctx context.Context, url string) ([]byte, error) {
	body, err := httpGet(ctx, url, ErrDownloadAsset)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	return readLimited(body)
}
