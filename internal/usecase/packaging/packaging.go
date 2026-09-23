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
	"github.com/device-management-toolkit/console/internal/entity/github"
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

	fetched, err := getReleases(ctx, releasesURL(s.githubBase, s.cfg.RPCRepo))
	if err != nil {
		if s.cfg.LocalDir == "" {
			return nil, err
		}

		s.l.Warn("github fetch failed, falling back to local dir: %v", err)

		return listLocalReleases(s.cfg.LocalDir)
	}

	releases := filterReleases(fetched)

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

	var (
		releases   []github.Release
		releaseErr error
	)

	if !s.cfg.DisableFetch {
		releases, releaseErr = getReleases(ctx, releasesURL(s.githubBase, s.cfg.RPCRepo))
	}

	for _, goos := range oses {
		data, assetName, err := s.resolveAsset(ctx, releases, releaseErr, req.Version, goos, req.Arch)
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

// fetchOnline downloads the matching asset from the fetched GitHub releases.
func fetchOnline(ctx context.Context, releases []github.Release, version, goos, arch string) (data []byte, assetName string, err error) {
	assetURL, name, found := findAsset(releases, version, goos, arch)
	if !found {
		return nil, "", fmt.Errorf("%w: version=%s os=%s arch=%s", ErrAssetNotFound, version, goos, arch)
	}

	data, err = downloadAsset(ctx, assetURL)
	if err != nil {
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

// resolveAsset downloads the asset from the fetched releases, falling back to the
// local directory when configured; with fetching disabled only the local directory is read.
func (s *Service) resolveAsset(ctx context.Context, releases []github.Release, releaseErr error, version, goos, arch string) (data []byte, assetName string, err error) {
	if s.cfg.DisableFetch {
		return findLocalAsset(s.cfg.LocalDir, version, goos, arch)
	}

	onlineErr := releaseErr
	if onlineErr == nil {
		data, assetName, onlineErr = fetchOnline(ctx, releases, version, goos, arch)
		if onlineErr == nil {
			return data, assetName, nil
		}
	}

	if s.cfg.LocalDir != "" {
		s.l.Warn("asset not available online, trying local dir: %v", onlineErr)

		return findLocalAsset(s.cfg.LocalDir, version, goos, arch)
	}

	return nil, "", onlineErr
}

// safeFilenamePart replaces characters unsafe in a Content-Disposition filename with a hyphen.
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

// validateVersion rejects a version that is not a single safe path element under LocalDir.
func validateVersion(version string) error {
	if version == "." || strings.ContainsAny(version, `/\`) || strings.Contains(version, "..") || version != filepath.Base(version) {
		return fmt.Errorf("%w: %q", ErrUnsafeVersion, version)
	}

	return nil
}

// findLocalAsset returns the matching asset file under <dir>/<version>/.
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
			b, readErr := os.ReadFile(filepath.Join(versionDir, e.Name()))
			if readErr != nil {
				return nil, "", fmt.Errorf("read local asset: %w", readErr)
			}

			return b, e.Name(), nil
		}
	}

	return nil, "", fmt.Errorf("%w: version=%s os=%s arch=%s", ErrAssetNotFound, version, goos, arch)
}

// buildConfigInputs resolves the server URLs and, for token auth, mints the JWT
// that renderConfig writes into config.yaml.
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
		// The OIDC verifier rejects Console-signed tokens.
		if s.cfg.ClientID != "" {
			return configInputs{}, ErrTokenModeUnsupported
		}

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
