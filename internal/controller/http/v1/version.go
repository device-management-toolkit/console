package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity/github"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var (
	ErrGithub        = consoleerrors.CreateConsoleError("LatestReleaseHandler")
	ErrFailedToFetch = errors.New("repositoryError")
)

type VersionRoute struct {
	Config *config.Config
}

// NewVersionRoute creates a new version route
func NewVersionRoute(configData *config.Config) *VersionRoute {
	return &VersionRoute{
		Config: configData,
	}
}

func RepositoryError(status string) error {
	return fmt.Errorf("failed to fetch latest release: %w: %s", ErrFailedToFetch, status)
}

// FetchLatestRelease fetches the latest release information from GitHub API
func (vr VersionRoute) FetchLatestRelease(c *gin.Context, repo string) (*github.Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	client := &http.Client{}

	req2, _ := http.NewRequestWithContext(c, http.MethodGet, url, http.NoBody)

	resp, err := client.Do(req2)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrGithub.Wrap("FetchLatestRelease", "http.Get", RepositoryError(resp.Status))
	}

	var release github.Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// LatestReleaseHandler is the Gin handler function to check for the latest release
func (vr VersionRoute) LatestReleaseHandler(c *gin.Context) {
	repo := vr.Config.App.Repo

	release, err := vr.FetchLatestRelease(c, repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"current": vr.Config.App.Version,
			"error":   err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current": vr.Config.App.Version,
		"latest": map[string]interface{}{
			"tag_name":     release.TagName,
			"name":         release.Name,
			"body":         release.Body,
			"prerelease":   release.Prerelease,
			"created_at":   release.CreatedAt,
			"published_at": release.PublishedAt,
			"html_url":     release.HTMLURL,
			"author":       release.Author,
			"assets":       release.Assets,
		},
	})
}
