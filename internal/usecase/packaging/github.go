package packaging

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/device-management-toolkit/console/internal/entity/github"
)

var assetRe = regexp.MustCompile(`(?i)_(Linux|Windows|Darwin)_([A-Za-z0-9_]+?)\.(?:tar\.gz|zip)$`)

const minSupportedMajor = 3

// parseAsset extracts a normalized os ("linux"/"windows"/"darwin") and arch
// token from an rpc-go release asset filename. ok is false for non-build assets.
func parseAsset(filename string) (os, arch string, ok bool) {
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
		if os, arch, ok := parseAsset(a.Name); ok {
			out = append(out, dtoAsset{os: os, arch: arch, name: a.Name, url: a.BrowserDownloadURL})
		}
	}

	return out
}
