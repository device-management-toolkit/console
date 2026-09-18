package packaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/entity/github"
)

var assetRe = regexp.MustCompile(`(?i)_(Linux|Windows|Darwin)_([A-Za-z0-9_]+?)\.(?:tar\.gz|zip|exe)$`)

// releasesURL builds the GitHub API releases URL for the given repo.
// base is overridable in tests (e.g. an httptest.Server URL).
func releasesURL(base, repo string) string {
	return fmt.Sprintf("%s/repos/%s/releases?per_page=%d", base, repo, maxReleases)
}

const minSupportedMajor = 3

// maxReleases caps how many releases are offered. Only the newest few are ever
// useful for a fresh activation, and asking GitHub for one short page keeps the
// call single-request — no pagination to walk.
const maxReleases = 5

// parseAsset extracts a normalized os ("linux"/"windows"/"darwin") and arch
// token from an rpc-go release asset filename. ok is false for non-build assets.
func parseAsset(filename string) (goos, arch string, ok bool) {
	m := assetRe.FindStringSubmatch(filename)
	if m == nil {
		return "", "", false
	}

	return strings.ToLower(m[1]), m[2], true
}

// isV3OrAbove reports whether a release tag is semver major >= 3 (betas count).
func isV3OrAbove(tag string) bool {
	t := strings.TrimPrefix(strings.TrimSpace(tag), "v")

	dot := strings.IndexByte(t, '.')
	if dot < 0 {
		return false
	}

	major, err := strconv.Atoi(t[:dot])
	if err != nil {
		return false
	}

	return major >= minSupportedMajor
}

// dtoAsset is the internal asset record (carries the download url/name for BuildPackage).
type dtoAsset struct {
	os, arch, name, url string
}

// toReleaseAssets maps github assets to internal records, keeping only parseable builds.
func toReleaseAssets(assets []github.Asset) []dtoAsset {
	out := make([]dtoAsset, 0, len(assets))

	for _, a := range assets {
		if assetOS, arch, ok := parseAsset(a.Name); ok {
			out = append(out, dtoAsset{os: assetOS, arch: arch, name: a.Name, url: a.BrowserDownloadURL})
		}
	}

	return out
}

// ErrFetchReleases indicates the GitHub releases request did not return 200.
var ErrFetchReleases = errors.New("failed to fetch releases")

// getReleases GETs a GitHub releases list URL and returns the raw release slice.
func getReleases(ctx context.Context, url string) ([]github.Release, error) {
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
		return nil, fmt.Errorf("%w: %s", ErrFetchReleases, resp.Status)
	}

	var releases []github.Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxArchiveBytes)).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

// listReleasesFrom GETs a GitHub releases list URL and returns the v3+ releases.
func listReleasesFrom(ctx context.Context, url string) ([]dto.RPCRelease, error) {
	releases, err := getReleases(ctx, url)
	if err != nil {
		return nil, err
	}

	return filterReleases(releases), nil
}

// filterReleases keeps the newest maxReleases v3+ releases and maps them to the
// UI DTO shape.
func filterReleases(releases []github.Release) []dto.RPCRelease {
	out := make([]dto.RPCRelease, 0, len(releases))

	for i := range releases {
		if len(out) == maxReleases {
			break
		}

		r := &releases[i]

		if !isV3OrAbove(r.TagName) {
			continue
		}

		internal := toReleaseAssets(r.Assets)
		assets := make([]dto.RPCAsset, 0, len(internal))

		for _, a := range internal {
			assets = append(assets, dto.RPCAsset{OS: a.os, Arch: a.arch})
		}

		out = append(out, dto.RPCRelease{Version: r.TagName, Assets: assets})
	}

	return out
}

// listLocalReleases scans an offline directory laid out as <dir>/<version>/<asset files>
// and returns the newest maxReleases v3+ versions.
func listLocalReleases(dir string) ([]dto.RPCRelease, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]dto.RPCRelease, 0, len(entries))

	for _, e := range entries {
		if !e.IsDir() || !isV3OrAbove(e.Name()) {
			continue
		}

		version := e.Name()

		files, err := os.ReadDir(filepath.Join(dir, version))
		if err != nil {
			return nil, err
		}

		assets := make([]dto.RPCAsset, 0, len(files))

		for _, f := range files {
			if assetOS, arch, ok := parseAsset(f.Name()); ok {
				assets = append(assets, dto.RPCAsset{OS: assetOS, Arch: arch})
			}
		}

		if len(assets) > 0 {
			out = append(out, dto.RPCRelease{Version: version, Assets: assets})
		}
	}

	slices.SortFunc(out, func(a, b dto.RPCRelease) int {
		return semver.Compare(canonicalTag(b.Version), canonicalTag(a.Version))
	})

	if len(out) > maxReleases {
		out = out[:maxReleases]
	}

	return out, nil
}

// canonicalTag adds the "v" prefix semver.Compare requires.
func canonicalTag(tag string) string {
	return "v" + strings.TrimPrefix(tag, "v")
}
