package packaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/entity/github"
)

var assetRe = regexp.MustCompile(`(?i)_(Linux|Windows|Darwin)_([A-Za-z0-9_]+?)\.(?:tar\.gz|zip)$`)

const minSupportedMajor = 3

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

// listReleasesFrom GETs a GitHub releases list URL and returns the v3+ releases.
func listReleasesFrom(ctx context.Context, url string) ([]dto.RpcRelease, error) {
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
		return nil, fmt.Errorf("%w: %s", ErrFetchReleases, resp.Status)
	}

	var releases []github.Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return filterReleases(releases), nil
}

// filterReleases keeps v3+ releases and maps them to the UI DTO shape.
func filterReleases(releases []github.Release) []dto.RpcRelease {
	out := make([]dto.RpcRelease, 0, len(releases))

	for i := range releases {
		r := &releases[i]

		if !isV3OrAbove(r.TagName) {
			continue
		}

		internal := toReleaseAssets(r.Assets)
		assets := make([]dto.RpcAsset, 0, len(internal))

		for _, a := range internal {
			assets = append(assets, dto.RpcAsset{OS: a.os, Arch: a.arch})
		}

		out = append(out, dto.RpcRelease{Version: r.TagName, Assets: assets})
	}

	return out
}

// listLocalReleases scans an offline directory laid out as <dir>/<version>/<asset files>.
func listLocalReleases(dir string) ([]dto.RpcRelease, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]dto.RpcRelease, 0, len(entries))

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		version := e.Name()

		files, err := os.ReadDir(filepath.Join(dir, version))
		if err != nil {
			return nil, err
		}

		assets := make([]dto.RpcAsset, 0, len(files))

		for _, f := range files {
			if assetOS, arch, ok := parseAsset(f.Name()); ok {
				assets = append(assets, dto.RpcAsset{OS: assetOS, Arch: arch})
			}
		}

		if len(assets) > 0 {
			out = append(out, dto.RpcRelease{Version: version, Assets: assets})
		}
	}

	return out, nil
}
