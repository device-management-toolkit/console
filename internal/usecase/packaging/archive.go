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
	"path"
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

// httpClient is a package-level HTTP client with a bounded timeout used for all
// outbound requests (releases list and asset downloads).
var httpClient = &http.Client{Timeout: httpTimeout} //nolint:gochecknoglobals // package-level singleton is intentional: shared across all requests to allow connection reuse

var (
	// ErrBinaryNotFound indicates the archive had no rpc/rpc.exe entry.
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

// extractBinary returns the rpc binary from a .tar.gz, .zip, or bare .exe asset.
func extractBinary(data []byte, assetName string) (name string, content []byte, err error) {
	switch lower := strings.ToLower(assetName); {
	case strings.HasSuffix(lower, ".exe"):
		return binaryNameWin, data, nil
	case strings.HasSuffix(lower, ".zip"):
		return extractFromZip(data)
	default:
		return extractFromTarGz(data)
	}
}

// nonBinaryExts are extensions carried by the metadata files a release ships
// beside the binary. An entry bearing one is never the binary, whatever its name,
// which keeps rpc_checksums.txt from being extracted and packaged as rpc.
var nonBinaryExts = map[string]bool{ //nolint:gochecknoglobals // package-level lookup table, read-only
	".txt":    true,
	".md":     true,
	".sha256": true,
	".sig":    true,
	".asc":    true,
	".pem":    true,
	".json":   true,
	".yaml":   true,
	".yml":    true,
	".log":    true,
}

// isBinaryEntry matches rpc, rpc.exe, and platform-suffixed names like rpc_linux_x64.
// A denylist rather than a required extension: goreleaser layouts embed version
// numbers, so "rpc_3.2.1_linux_x86_64" has a dotted tail that is not an extension.
func isBinaryEntry(name string) bool {
	base := path.Base(name)

	if base == binaryName || base == binaryNameWin {
		return true
	}

	return strings.HasPrefix(base, binaryName+"_") && !nonBinaryExts[strings.ToLower(path.Ext(base))]
}

// packagedBinaryName is the name the binary gets inside the downloaded zip.
func packagedBinaryName(entry string) string {
	if strings.HasSuffix(strings.ToLower(entry), ".exe") {
		return binaryNameWin
	}

	return binaryName
}

func extractFromTarGz(data []byte) (name string, content []byte, err error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return "", nil, fmt.Errorf("tar: %w", err)
		}

		if hdr.Typeflag == tar.TypeReg && isBinaryEntry(hdr.Name) {
			content, err := readLimited(tr)
			if err != nil {
				return "", nil, err
			}

			return packagedBinaryName(hdr.Name), content, nil
		}
	}

	return "", nil, ErrBinaryNotFound
}

func extractFromZip(data []byte) (name string, content []byte, err error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", nil, fmt.Errorf("zip: %w", err)
	}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !isBinaryEntry(f.Name) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", nil, fmt.Errorf("zip open: %w", err)
		}

		content, err := readLimited(rc)
		rc.Close()

		if err != nil {
			return "", nil, err
		}

		return packagedBinaryName(f.Name), content, nil
	}

	return "", nil, ErrBinaryNotFound
}

// zipEntry is one binary placed in the downloadable zip.
type zipEntry struct {
	name    string
	content []byte
}

// buildZip assembles the downloadable zip containing the binaries and config.yaml.
func buildZip(binaries []zipEntry, configYAML []byte) ([]byte, error) {
	var buf bytes.Buffer

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

// downloadAsset fetches an asset's bytes over HTTP (used for the online path).
func downloadAsset(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrDownloadAsset, resp.Status)
	}

	return readLimited(resp.Body)
}
