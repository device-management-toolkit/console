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
)

const (
	binaryName      = "rpc"
	binaryNameWin   = "rpc.exe"
	configFileName  = "config.yaml"
	binaryFileMode  = 0o755
	maxArchiveBytes = 200 << 20 // 200 MiB cap to guard against decompression bombs
)

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

// extractBinary pulls the rpc/rpc.exe binary out of a .tar.gz or .zip asset.
func extractBinary(data []byte, assetName string) (name string, content []byte, err error) {
	if strings.HasSuffix(strings.ToLower(assetName), ".zip") {
		return extractFromZip(data)
	}

	return extractFromTarGz(data)
}

func isBinaryEntry(name string) bool {
	base := path.Base(name)

	return base == binaryName || base == binaryNameWin
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

		if isBinaryEntry(hdr.Name) {
			content, err := readLimited(tr)
			if err != nil {
				return "", nil, err
			}

			return path.Base(hdr.Name), content, nil
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
		if !isBinaryEntry(f.Name) {
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

		return path.Base(f.Name), content, nil
	}

	return "", nil, ErrBinaryNotFound
}

// buildZip assembles the downloadable zip containing the binary and config.yaml.
func buildZip(binFileName string, binary, configYAML []byte) ([]byte, error) {
	var buf bytes.Buffer

	zw := zip.NewWriter(&buf)

	binHeader := &zip.FileHeader{Name: binFileName, Method: zip.Deflate}
	binHeader.SetMode(binaryFileMode)

	bw, err := zw.CreateHeader(binHeader)
	if err != nil {
		return nil, fmt.Errorf("zip create binary: %w", err)
	}

	if _, err := bw.Write(binary); err != nil {
		return nil, fmt.Errorf("zip write binary: %w", err)
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrDownloadAsset, resp.Status)
	}

	return readLimited(resp.Body)
}
