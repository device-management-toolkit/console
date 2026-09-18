package packaging

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity/github"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestParseChecksums(t *testing.T) {
	t.Parallel()

	body := []byte("" +
		"aaaa  rpc_Linux_x86_64.tar.gz\n" +
		"bbbb *rpc_Windows_x86_64.zip\n" +
		"garbage line with three fields here\n")

	tests := []struct {
		name  string
		asset string
		want  string
		ok    bool
	}{
		{"plain entry", "rpc_Linux_x86_64.tar.gz", "aaaa", true},
		{"binary-mode star is stripped", "rpc_Windows_x86_64.zip", "bbbb", true},
		{"absent asset", "rpc_Darwin_arm64.tar.gz", "", false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseChecksums(body, tt.asset)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("parseChecksums() = %q, %v; want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// newChecksumService returns a Service plus a release list whose checksums asset
// is served by an httptest server carrying the supplied body.
func newChecksumService(t *testing.T, checksumBody string, withChecksumAsset bool) (*Service, []github.Release) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, checksumBody)
	}))
	t.Cleanup(srv.Close)

	assets := []github.Asset{{Name: "rpc_Linux_x86_64.tar.gz", BrowserDownloadURL: srv.URL + "/asset"}}
	if withChecksumAsset {
		assets = append(assets, github.Asset{Name: "rpc_checksums.txt", BrowserDownloadURL: srv.URL + "/sums"})
	}

	return New(newTestConfig(""), logger.New("error")), []github.Release{
		{TagName: "v3.0.1", Assets: assets},
	}
}

func TestVerifyChecksumMatches(t *testing.T) {
	t.Parallel()

	data := []byte("binary-bytes")
	sum := sha256.Sum256(data)
	body := hex.EncodeToString(sum[:]) + "  rpc_Linux_x86_64.tar.gz\n"

	svc, releases := newChecksumService(t, body, true)

	if err := svc.verifyChecksum(context.Background(), releases, "v3.0.1", "rpc_Linux_x86_64.tar.gz", data); err != nil {
		t.Fatalf("verifyChecksum() = %v, want nil", err)
	}
}

func TestVerifyChecksumMismatchFails(t *testing.T) {
	t.Parallel()

	body := "00000000000000000000000000000000000000000000000000000000deadbeef  rpc_Linux_x86_64.tar.gz\n"

	svc, releases := newChecksumService(t, body, true)

	err := svc.verifyChecksum(context.Background(), releases, "v3.0.1", "rpc_Linux_x86_64.tar.gz", []byte("binary-bytes"))
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("verifyChecksum() = %v, want ErrChecksumMismatch", err)
	}
}

// A release that ships no checksums asset is unverified, not broken: the build
// proceeds so an older release without one stays downloadable.
func TestVerifyChecksumNoAssetProceeds(t *testing.T) {
	t.Parallel()

	svc, releases := newChecksumService(t, "", false)

	if err := svc.verifyChecksum(context.Background(), releases, "v3.0.1", "rpc_Linux_x86_64.tar.gz", []byte("x")); err != nil {
		t.Fatalf("verifyChecksum() = %v, want nil", err)
	}
}

// A checksums file that records no line for this asset is likewise unverified.
func TestVerifyChecksumAssetAbsentFromFileProceeds(t *testing.T) {
	t.Parallel()

	svc, releases := newChecksumService(t, "aaaa  some-other-asset.tar.gz\n", true)

	if err := svc.verifyChecksum(context.Background(), releases, "v3.0.1", "rpc_Linux_x86_64.tar.gz", []byte("x")); err != nil {
		t.Fatalf("verifyChecksum() = %v, want nil", err)
	}
}
