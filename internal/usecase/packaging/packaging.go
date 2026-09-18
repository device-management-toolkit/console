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

	schemeHTTP  = "http"
	schemeHTTPS = "https"

	// osBoth requests the Windows and Linux builds in one package.
	osBoth = "both"
)

// listenerScheme returns the URL scheme the HTTP listener serves.
func listenerScheme(tlsEnabled bool) string {
	if tlsEnabled {
		return schemeHTTPS
	}

	return schemeHTTP
}

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

// ListVersions returns the available rpc-go releases: GitHub's first, then any
// local-only versions. With fetching disabled only the local directory is read;
// if GitHub fails the local directory is the fallback.
func (s *Service) ListVersions(ctx context.Context) ([]dto.RPCRelease, error) {
	if s.cfg.DisableFetch {
		return listLocalReleases(s.cfg.LocalDir)
	}

	releases, err := listReleasesFrom(ctx, releasesURL(s.githubBase, s.cfg.RPCRepo))
	if err != nil {
		if s.cfg.LocalDir == "" {
			return nil, err
		}

		s.l.Warn("github fetch failed, falling back to local dir: %v", err)

		return listLocalReleases(s.cfg.LocalDir)
	}

	if s.cfg.LocalDir == "" {
		return releases, nil
	}

	local, err := listLocalReleases(s.cfg.LocalDir)
	if err != nil {
		s.l.Warn("read local dir, listing github releases only: %v", err)

		return releases, nil
	}

	return mergeReleases(releases, local), nil
}

// BuildPackage resolves the requested rpc-go binary, renders a config.yaml, and
// returns a zip reader together with a suggested filename. tenantID scopes the
// generated config to the caller's tenant.
func (s *Service) BuildPackage(ctx context.Context, req dto.PackageRequest, tenantID string) (io.Reader, string, error) {
	oses := packageOSes(req.OS)
	binaries := make([]zipEntry, 0, len(oses))

	for _, goos := range oses {
		data, assetName, err := s.resolveAsset(ctx, req.Version, goos, req.Arch)
		if err != nil {
			return nil, "", fmt.Errorf("resolve asset: %w", err)
		}

		binName, binary, err := extractBinary(data, assetName)
		if err != nil {
			return nil, "", fmt.Errorf("extract binary: %w", err)
		}

		binaries = append(binaries, zipEntry{name: binName, content: binary})
	}

	inputs, err := s.buildConfigInputs(req, tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("build config inputs: %w", err)
	}

	cfgYAML, err := renderConfig(req, inputs)
	if err != nil {
		return nil, "", fmt.Errorf("render config: %w", err)
	}

	zipBytes, err := buildZip(binaries, cfgYAML)
	if err != nil {
		return nil, "", fmt.Errorf("build zip: %w", err)
	}

	filename := fmt.Sprintf("rpc-%s-%s-%s.zip", safeFilenamePart(req.Command), safeFilenamePart(req.OS), safeFilenamePart(req.Arch))

	return bytes.NewReader(zipBytes), filename, nil
}

// fetchOnline resolves the asset from GitHub and verifies it against the
// checksums published with the release. A nil error means data is verified.
func (s *Service) fetchOnline(ctx context.Context, version, goos, arch string) (data []byte, assetName string, err error) {
	releases, err := getReleases(ctx, releasesURL(s.githubBase, s.cfg.RPCRepo))
	if err != nil {
		return nil, "", err
	}

	assetURL, name, found := findAsset(releases, version, goos, arch)
	if !found {
		return nil, "", fmt.Errorf("%w: version=%s os=%s arch=%s", ErrAssetNotFound, version, goos, arch)
	}

	data, err = downloadAsset(ctx, assetURL)
	if err != nil {
		return nil, "", err
	}

	if err := s.verifyChecksum(ctx, releases, version, name, data); err != nil {
		return nil, "", err
	}

	return data, name, nil
}

// packageOSes lists the builds a request packages; "both" means Windows and Linux.
func packageOSes(goos string) []string {
	if goos == osBoth {
		return []string{"windows", "linux"}
	}

	return []string{goos}
}

// resolveAsset tries to fetch the asset from GitHub; on failure it falls back to
// the local cache directory when configured. With fetching disabled it reads the
// local directory only. Returns the raw bytes and asset name.
func (s *Service) resolveAsset(ctx context.Context, version, goos, arch string) (data []byte, assetName string, err error) {
	if s.cfg.DisableFetch {
		return findLocalAsset(s.cfg.LocalDir, version, goos, arch)
	}

	data, assetName, onlineErr := s.fetchOnline(ctx, version, goos, arch)
	if onlineErr == nil {
		return data, assetName, nil
	}

	// A mismatch means the bytes served were wrong, not absent — falling back to
	// a local copy would hide that, so it is reported.
	if errors.Is(onlineErr, ErrChecksumMismatch) {
		return nil, "", onlineErr
	}

	if s.cfg.LocalDir != "" {
		s.l.Warn("asset not available online, trying local dir: %v", onlineErr)

		return findLocalAsset(s.cfg.LocalDir, version, goos, arch)
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
// The base URL is req.ServerURL when the caller supplied one, else a URL
// constructed from the HTTP host and port.
// When auth mode is "token" and auth is enabled, a short-lived JWT is minted
// from the configured JWT key.
func (s *Service) buildConfigInputs(req dto.PackageRequest, tenantID string) (configInputs, error) {
	base := strings.TrimRight(req.ServerURL, "/")

	if base == "" {
		// Best-effort default for callers that do not supply a server URL.
		host := s.cfg.Host
		if host == "" {
			host = "localhost"
		}

		base = fmt.Sprintf("%s://%s:%s", listenerScheme(s.cfg.TLS.Enabled), host, s.cfg.Port)
	}

	// Without a configured cert the listener serves a generated self-signed one,
	// which rpc-go cannot chain to a trusted root.
	skipCertCheck := s.cfg.TLS.Enabled && s.cfg.TLS.CertFile == ""

	in := configInputs{
		AuthEndpoint:    base + "/api/v1/authorize",
		DevicesEndpoint: base + "/api/v1/devices",
		ExportBase:      base,
		TenantID:        tenantID,
		SkipCertCheck:   skipCertCheck,
	}

	// With auth disabled the server accepts requests without a token, so none is minted.
	if req.Auth.Mode == authModeToken && !s.cfg.Disabled {
		ttl, ttlErr := resolveTokenTTL(req.TokenTTL, s.cfg.MaxTokenTTL)
		if ttlErr != nil {
			return configInputs{}, ttlErr
		}

		tok, mintErr := mintToken(s.cfg.JWTKey, ttl)
		if mintErr != nil {
			return configInputs{}, fmt.Errorf("mint token: %w", mintErr)
		}

		in.AuthToken = tok
	}

	return in, nil
}
