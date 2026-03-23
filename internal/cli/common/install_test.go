package common

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v78/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	vrsrgithub "github.com/stepbeta/vrsr/internal/github"
)

type mockReposService struct {
	releases []*github.RepositoryRelease
}

func (m *mockReposService) ListReleases(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) {
	return m.releases, &github.Response{}, nil
}

func (m *mockReposService) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, *github.Response, error) {
	return nil, &github.Response{}, nil
}

func (m *mockReposService) DownloadReleaseAsset(ctx context.Context, owner, repo string, assetID int64, httpClient *http.Client) (io.ReadCloser, string, error) {
	return nil, "", nil
}

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name           string
		releases       []*github.RepositoryRelease
		expectedLatest string
		expectedError  bool
	}{
		{
			name: "returns highest semver",
			releases: []*github.RepositoryRelease{
				{TagName: github.Ptr("v1.0.0")},
				{TagName: github.Ptr("v1.2.3")},
				{TagName: github.Ptr("v1.1.0")},
			},
			expectedLatest: "v1.2.3",
			expectedError:  false,
		},
		{
			name: "handles complex semver versions",
			releases: []*github.RepositoryRelease{
				{TagName: github.Ptr("v1.0.0")},
				{TagName: github.Ptr("v2.0.0-alpha")},
				{TagName: github.Ptr("v2.0.0-beta")},
				{TagName: github.Ptr("v2.0.0-rc.1")},
				{TagName: github.Ptr("v2.0.0")},
			},
			expectedLatest: "v2.0.0",
			expectedError:  false,
		},
		{
			name:           "returns error when no releases",
			releases:       []*github.RepositoryRelease{},
			expectedLatest: "",
			expectedError:  true,
		},
		{
			name: "handles non-semver tags gracefully",
			releases: []*github.RepositoryRelease{
				{TagName: github.Ptr("not-a-version")},
				{TagName: github.Ptr("v1.0.0")},
			},
			expectedLatest: "v1.0.0",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepos := &mockReposService{releases: tt.releases}
			mockGh := vrsrgithub.GithubHelper{
				Repos: mockRepos,
			}

			result, err := getLatestVersion("testtool", vrsrgithub.RepoConfDef{}, mockGh)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expectedLatest {
					t.Errorf("expected %s, got %s", tt.expectedLatest, result)
				}
			}
		})
	}
}

func TestInstallCommands_EarlyReturnWhenInUseOrInstalled(t *testing.T) {
	// setup temp dirs
	td := t.TempDir()
	vrsPath := filepath.Join(td, "versions")
	binPath := filepath.Join(td, "bin")

	// create dirs
	if err := os.MkdirAll(filepath.Join(vrsPath, "mytool"), 0o755); err != nil {
		t.Fatalf("failed to create vrs dir: %v", err)
	}

	// create a fake installed binary
	installedFile := filepath.Join(vrsPath, "mytool", "mytool-1.2.3")
	if err := os.WriteFile(installedFile, []byte("hello"), 0o755); err != nil {
		t.Fatalf("failed to create installed file: %v", err)
	}

	// create bin dir and symlink to indicate it's in use
	if err := os.MkdirAll(binPath, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	linkTarget := installedFile
	symlink := filepath.Join(binPath, "mytool")
	if err := os.Symlink(linkTarget, symlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// set viper paths
	viper.Set("vrs-path", vrsPath)
	viper.Set("bin-path", binPath)

	// calling install Github should return early (tool already in use)
	if err := install(&cobra.Command{}, "1.2.3", "mytool", vrsrgithub.RepoConfDef{}, InstallGitHubCmd, false); err != nil {
		t.Fatalf("install Github expected nil error when tool in use, got: %v", err)
	}

	// remove symlink to test IsToolInstalled early return
	if err := os.Remove(symlink); err != nil {
		t.Fatalf("failed to remove symlink: %v", err)
	}

	// calling install Download should return early (tool already installed)
	if err := install(&cobra.Command{}, "1.2.3", "mytool", vrsrgithub.RepoConfDef{}, InstallDownloadCmd, false); err != nil {
		t.Fatalf("install Download expected nil error when tool installed, got: %v", err)
	}
}
