package packaging

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/device-management-toolkit/console/internal/entity/github"
)

// ErrChecksumMismatch indicates a downloaded asset did not match the SHA256
// published alongside it in the release.
var ErrChecksumMismatch = errors.New("asset checksum mismatch")

const (
	// checksumsAssetMarker identifies the goreleaser checksums file within a release.
	checksumsAssetMarker = "checksums"
	// checksumFields is the column count of a "<sha256>  <filename>" line.
	checksumFields = 2
)

// findChecksumsAsset returns the download URL of the release's checksums file.
// Releases that publish no such asset return ok false.
func findChecksumsAsset(releases []github.Release, version string) (url string, ok bool) {
	for i := range releases {
		if releases[i].TagName != version {
			continue
		}

		for _, a := range releases[i].Assets {
			if strings.Contains(strings.ToLower(a.Name), checksumsAssetMarker) {
				return a.BrowserDownloadURL, true
			}
		}
	}

	return "", false
}

// parseChecksums finds the SHA256 recorded for assetName in a goreleaser-style
// checksums file, whose lines are "<hex sha256>  <filename>". ok is false when
// the file records no entry for that asset.
func parseChecksums(body []byte, assetName string) (sum string, ok bool) {
	for line := range strings.SplitSeq(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) != checksumFields {
			continue
		}

		// The filename column may carry a leading "*" marking binary mode.
		if strings.TrimPrefix(fields[1], "*") == assetName {
			return strings.ToLower(fields[0]), true
		}
	}

	return "", false
}

// verifyChecksum checks data against the SHA256 published for assetName in the
// release. A release with no checksums file, or one that records no entry for
// this asset, is reported as unverified rather than failing the build; only a
// genuine mismatch is an error.
func (s *Service) verifyChecksum(ctx context.Context, releases []github.Release, version, assetName string, data []byte) error {
	url, ok := findChecksumsAsset(releases, version)
	if !ok {
		s.l.Warn("release %s publishes no checksums asset, skipping verification", version)

		return nil
	}

	body, err := downloadAsset(ctx, url)
	if err != nil {
		s.l.Warn("could not fetch checksums for %s, skipping verification: %v", version, err)

		return nil
	}

	want, ok := parseChecksums(body, assetName)
	if !ok {
		s.l.Warn("checksums for %s record no entry for %s, skipping verification", version, assetName)

		return nil
	}

	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("%w: %s: want %s, got %s", ErrChecksumMismatch, assetName, want, hex.EncodeToString(got[:]))
	}

	return nil
}
