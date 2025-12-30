package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v78/github"
	"github.com/schollz/progressbar/v3"
	"github.com/stepbeta/vrsr/internal/utils"
)

var errReleaseNotFound = errors.New("release not found")

type GithubHelper struct {
	Client *github.Client
	Repos  repositoriesService
}

// repositoriesService defines the subset of github repository methods used by this helper.
type repositoriesService interface {
	ListReleases(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error)
	GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, *github.Response, error)
	DownloadReleaseAsset(ctx context.Context, owner, repo string, assetID int64, httpClient *http.Client) (io.ReadCloser, string, error)
}

func New(client *github.Client) GithubHelper {
	if client == nil {
		client = github.NewClient(nil)
	}
	// Optional: Use token for higher rate limits:
	// - anonymous: 60 calls per hour
	// - authenticated: 5,000 calls per hour
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		client = client.WithAuthToken(token)
	}
	return GithubHelper{Client: client, Repos: client.Repositories}
}

type RepoConfDef struct {
	Org         string
	Repo        string
	Zipped      bool
	DownloadURL string
}

type FetchOptions struct {
	IncludeDevel bool
	Limit        int
	Force        bool
	RepoConf     RepoConfDef
}

// FetchAllReleases fetches all releases from the GitHub repository
func (gh *GithubHelper) FetchAllReleases(tool string, opts FetchOptions) (utils.ReleasesData, error) {
	ctx := context.Background()

	if !opts.Force {
		cacheData, err := utils.ReadFromCache(tool, opts.Limit)
		if err == nil && cacheData.Releases != nil && len(cacheData.Releases) > 0 {
			return cacheData, nil
		}
	}

	totPages := 1
	bar := progressbar.NewOptions(totPages,
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetDescription("Downloading releases metadata..."),
		progressbar.OptionClearOnFinish(),
	)
	defer func() {
		_ = bar.Finish()
	}()

	var allReleases []*github.RepositoryRelease
	page := 1

	// we use max possible value in order to limit occurrence of rate-limiting
	limit := 100
	if opts.Limit < 100 {
		limit = opts.Limit
	}

pages:
	for {
		releases, resp, err := gh.Repos.ListReleases(ctx, opts.RepoConf.Org, opts.RepoConf.Repo, &github.ListOptions{
			Page:    page,
			PerPage: limit,
		})
		if err != nil {
			return utils.ReleasesData{}, err
		}
		if resp.LastPage > 1 && totPages != resp.LastPage {
			bar.ChangeMax(resp.LastPage)
		}

		for _, r := range releases {
			if opts.Limit > 0 && len(allReleases) >= opts.Limit {
				break pages
			}
			allReleases = append(allReleases, r)
		}

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
		_ = bar.Add(1)
	}

	utils.SaveToCache(tool, allReleases)

	return utils.ReleasesData{
		Timestamp: time.Now().UTC(),
		Releases:  allReleases,
	}, nil
}

// DownloadRelease downloads the specified release version to the given vrsPath
func (gh *GithubHelper) DownloadRelease(tool, version, vrsPath string, repo RepoConfDef) error {
	ctx := context.Background()
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetDescription("Downloading release metadata..."),
		progressbar.OptionClearOnFinish(),
	)
	defer func() {
		_ = bar.Finish()
	}()
	rel, _, err := gh.Repos.GetReleaseByTag(ctx, repo.Org, repo.Repo, version)
	if err != nil {
		return err
	}
	if rel == nil {
		return errReleaseNotFound
	}

	bar.Describe("Finding the right asset to download...")
	osAlias := strings.ToLower(runtime.GOOS)
	archAlias := strings.ToLower(runtime.GOARCH)

	relName := tool + "-" + osAlias + "-" + archAlias

	var asset *github.ReleaseAsset
	for _, a := range rel.Assets {
		if a == nil {
			continue
		}
		lname := strings.ToLower(a.GetName())
		if !strings.HasPrefix(lname, relName) {
			// not the right asset
			continue
		}
		if osAlias == "windows" && !strings.HasSuffix(lname, ".exe") {
			// windows binary must have .exe suffix
			continue
		}
		if osAlias != "windows" && len(strings.Split(lname, ".")) > 1 {
			// non-windows binaries should not have an extension
			continue
		}
		asset = a
	}
	if asset == nil {
		return errReleaseNotFound
	}

	// download asset using go-github helper (returns ReadCloser)
	bar.Describe("Downloading...")
	rc, _, err := gh.Repos.DownloadReleaseAsset(ctx, repo.Org, repo.Repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer func() {
		_ = rc.Close()
	}()

	// write to temp file then move (safer)
	finalPath := filepath.Join(vrsPath, tool)
	err = utils.EnsurePathExists(finalPath)
	if err != nil {
		return fmt.Errorf("error ensuring vrs path exists: %w", err)
	}
	tmpFile, err := os.CreateTemp(finalPath, tool+"-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	// Ensure cleanup if something goes wrong
	defer func() {
		if err != nil {
			_ = os.Remove(tmpFile.Name())
		}
	}()
	_, err = io.Copy(tmpFile, rc)
	if err1 := tmpFile.Close(); err == nil && err1 != nil {
		err = err1
	}
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return fmt.Errorf("failed to save download: %w", err)
	}

	// move to destination
	destPath := filepath.Join(finalPath, tool+"-"+version)
	if err := os.Rename(tmpFile.Name(), destPath); err != nil {
		return fmt.Errorf("failed to move downloaded file to destination: %w", err)
	}

	// make it executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission: %w", err)
	}
	return nil
}
