package packaging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	githubDefaultBase = "https://api.github.com"
)

// ErrAssetNotFound is returned when the requested asset cannot be found online or locally.
var ErrAssetNotFound = errors.New("asset not found")

// ErrUnsafeVersion is returned when req.Version contains path-traversal characters.
var ErrUnsafeVersion = errors.New("unsafe version: contains path separator or dot-dot")

// Service implements the Feature interface for building rpc-go download packages.
type Service struct {
	cfg        *config.Config
	l          logger.Interface
	githubBase string
}

// New constructs a Service with the default GitHub API base URL.
func New(cfg *config.Config, l logger.Interface) *Service {
	return &Service{
		cfg:        cfg,
		l:          l,
		githubBase: githubDefaultBase,
	}
}

// ListVersions returns the available rpc-go releases.
// If the GitHub fetch fails and a local cache directory is configured, it falls
// back to scanning the local directory.
func (s *Service) ListVersions(ctx context.Context) ([]dto.RPCRelease, error) {
	releases, err := listReleasesFrom(ctx, releasesURL(s.githubBase, s.cfg.RPCRepo))
	if err == nil {
		return releases, nil
	}

	if s.cfg.LocalDir != "" {
		s.l.Warn("github fetch failed, falling back to local dir: %v", err)

		return listLocalReleases(s.cfg.LocalDir)
	}

	return nil, err
}

// BuildPackage resolves the requested rpc-go binary, renders a config.yaml, and
// returns a zip reader together with a suggested filename.
func (s *Service) BuildPackage(ctx context.Context, req dto.PackageRequest) (io.Reader, string, error) {
	data, assetName, err := s.resolveAsset(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("resolve asset: %w", err)
	}

	binName, binary, err := extractBinary(data, assetName)
	if err != nil {
		return nil, "", fmt.Errorf("extract binary: %w", err)
	}

	inputs, err := s.buildConfigInputs(req)
	if err != nil {
		return nil, "", fmt.Errorf("build config inputs: %w", err)
	}

	cfgYAML, err := renderConfig(req, inputs)
	if err != nil {
		return nil, "", fmt.Errorf("render config: %w", err)
	}

	zipBytes, err := buildZip(binName, binary, cfgYAML)
	if err != nil {
		return nil, "", fmt.Errorf("build zip: %w", err)
	}

	filename := fmt.Sprintf("rpc-%s-%s-%s.zip", safeFilenamePart(req.Command), safeFilenamePart(req.OS), safeFilenamePart(req.Arch))

	return bytes.NewReader(zipBytes), filename, nil
}

// resolveAsset tries to fetch the asset from GitHub; on failure it falls back to
// the local cache directory when configured. Returns the raw bytes and asset name.
func (s *Service) resolveAsset(ctx context.Context, req dto.PackageRequest) (data []byte, assetName string, err error) {
	url := releasesURL(s.githubBase, s.cfg.RPCRepo)

	releases, onlineErr := getReleases(ctx, url)
	if onlineErr == nil {
		assetURL, name, found := findAsset(releases, req.Version, req.OS, req.Arch)
		if found {
			var dlData []byte

			dlData, onlineErr = downloadAsset(ctx, assetURL)
			if onlineErr == nil {
				return dlData, name, nil
			}
		} else {
			onlineErr = fmt.Errorf("%w: version=%s os=%s arch=%s", ErrAssetNotFound, req.Version, req.OS, req.Arch)
		}
	}

	if s.cfg.LocalDir != "" {
		s.l.Warn("asset not available online, trying local dir: %v", onlineErr)

		return findLocalAsset(s.cfg.LocalDir, req.Version, req.OS, req.Arch)
	}

	return nil, "", onlineErr
}

// safeFilenamePart keeps a filename component limited to safe characters.
// Any character that is not a letter, digit, dot, hyphen, or underscore is
// replaced with a hyphen, preventing injection into Content-Disposition headers.
func safeFilenamePart(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, s)
}

// validateVersion rejects any version string that contains a path separator,
// ".." segment, or is the single-dot current-directory marker, preventing
// directory traversal when the version is used to build a filesystem path
// under LocalDir.
func validateVersion(version string) error {
	if version == "." {
		return fmt.Errorf("%w: %q", ErrUnsafeVersion, version)
	}

	if strings.Contains(version, "/") || strings.Contains(version, "\\") || strings.Contains(version, "..") {
		return fmt.Errorf("%w: %q", ErrUnsafeVersion, version)
	}

	// filepath.Base strips any leading directory components; if the result differs
	// from the input the version encodes a path rather than a single element.
	if version != filepath.Base(version) {
		return fmt.Errorf("%w: %q", ErrUnsafeVersion, version)
	}

	return nil
}

// findLocalAsset looks up the matching asset file under <dir>/<version>/ and
// returns its contents. The version is validated to be a single, safe path
// element before any file operations are performed.
func findLocalAsset(dir, version, goos, arch string) (data []byte, assetName string, err error) {
	if err := validateVersion(version); err != nil {
		return nil, "", err
	}

	versionDir := filepath.Join(dir, version)

	entries, rdErr := os.ReadDir(versionDir)
	if rdErr != nil {
		return nil, "", fmt.Errorf("read local version dir: %w", rdErr)
	}

	for _, e := range entries {
		if assetOS, assetArch, ok := parseAsset(e.Name()); ok && assetOS == goos && assetArch == arch {
			assetPath := filepath.Join(versionDir, e.Name())
			// version is validated above to be a single path element with no separators
			// or dot-dot; combined with the trusted dir and an enumerated filename,
			// assetPath cannot escape dir.
			b, readErr := os.ReadFile(assetPath)
			if readErr != nil {
				return nil, "", fmt.Errorf("read local asset: %w", readErr)
			}

			return b, e.Name(), nil
		}
	}

	return nil, "", fmt.Errorf("%w: version=%s os=%s arch=%s", ErrAssetNotFound, version, goos, arch)
}

// buildConfigInputs resolves the configInputs for rendering the rpc-go config.yaml.
// When PublicURL is empty it constructs a base URL from the HTTP host and port.
// When auth mode is "token" a short-lived JWT is minted from the configured JWT key.
func (s *Service) buildConfigInputs(req dto.PackageRequest) (configInputs, error) {
	base := s.cfg.PublicURL
	if base == "" {
		// Fall back to a URL derived from the HTTP listener address. This is a
		// best-effort default for deployments that have not set CONSOLE_PUBLIC_URL.
		host := s.cfg.Host
		if host == "" {
			host = "localhost"
		}

		base = fmt.Sprintf("http://%s:%s", host, s.cfg.Port)
	}

	in := configInputs{
		AuthEndpoint:    base + "/api/v1/authorize",
		DevicesEndpoint: base + "/api/v1/devices",
		ExportBase:      base,
	}

	if req.Auth.Mode == authModeToken {
		tok, mintErr := mintToken(s.cfg.JWTKey)
		if mintErr != nil {
			return configInputs{}, fmt.Errorf("mint token: %w", mintErr)
		}

		in.AuthToken = tok
	}

	return in, nil
}
