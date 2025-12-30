package github

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gh "github.com/google/go-github/v78/github"
	"github.com/stepbeta/vrsr/internal/utils"
)

func TestNew_NoTokenAndWithToken(t *testing.T) {
	// preserve env
	orig := os.Getenv("GITHUB_TOKEN")
	defer func() {
		if err := os.Setenv("GITHUB_TOKEN", orig); err != nil {
			t.Fatalf("failed to restore GITHUB_TOKEN: %v", err)
		}
	}()

	// ensure absent
	_ = os.Unsetenv("GITHUB_TOKEN")
	g := New(nil)
	if g.Client == nil {
		t.Fatalf("expected non-nil client when no token set")
	}

	// set fake token
	_ = os.Setenv("GITHUB_TOKEN", "fake-token")
	g2 := New(nil)
	if g2.Client == nil {
		t.Fatalf("expected non-nil client when token set")
	}
}

func TestFetchAllReleases_UsesCacheAndLimit(t *testing.T) {
	td := t.TempDir()
	// ensure GetCachePath uses this home
	origHome := os.Getenv("HOME")
	defer func() {
		if err := os.Setenv("HOME", origHome); err != nil {
			t.Fatalf("failed to restore HOME: %v", err)
		}
	}()
	if err := os.Setenv("HOME", td); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	tool := "cachetool"

	// build fake releases
	rels := []*gh.RepositoryRelease{
		{TagName: gh.Ptr("v1.0.0")},
		{TagName: gh.Ptr("v1.1.0")},
		{TagName: gh.Ptr("v2.0.0")},
	}

	// save to cache (uses GetCachePath which will look under HOME)
	utils.SaveToCache(tool, rels)

	ghh := New(nil)
	// non-forced fetch should return cache
	data, err := ghh.FetchAllReleases(tool, FetchOptions{Force: false, Limit: 0, RepoConf: RepoConfDef{}})
	if err != nil {
		t.Fatalf("FetchAllReleases returned error: %v", err)
	}
	if data.Releases == nil || len(data.Releases) != len(rels) {
		t.Fatalf("expected %d cached releases, got %d", len(rels), len(data.Releases))
	}

	// test limit
	data2, err := ghh.FetchAllReleases(tool, FetchOptions{Force: false, Limit: 2, RepoConf: RepoConfDef{}})
	if err != nil {
		t.Fatalf("FetchAllReleases with limit returned error: %v", err)
	}
	if data2.Releases == nil || len(data2.Releases) != 2 {
		t.Fatalf("expected 2 cached releases with limit, got %d", len(data2.Releases))
	}

	// ensure cache file exists at expected path
	path, err := utils.GetCachePath(tool)
	if err != nil {
		t.Fatalf("GetCachePath error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected cache file at %s, stat error: %v", path, err)
	}

	// cleanup cache file explicitly
	_ = os.Remove(filepath.Clean(path))
}

type fakeReposForTest struct {
	releases []*gh.RepositoryRelease
}

func (f *fakeReposForTest) ListReleases(ctx context.Context, owner, repo string, opts *gh.ListOptions) ([]*gh.RepositoryRelease, *gh.Response, error) {
	return f.releases, &gh.Response{NextPage: 0, LastPage: 1}, nil
}

func (f *fakeReposForTest) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*gh.RepositoryRelease, *gh.Response, error) {
	for _, r := range f.releases {
		if r.GetTagName() == tag {
			return r, &gh.Response{}, nil
		}
	}
	return nil, nil, nil
}

func (f *fakeReposForTest) DownloadReleaseAsset(ctx context.Context, owner, repo string, assetID int64, httpClient *http.Client) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader("ok")), "", nil
}

func TestFetchAllReleases_ForcedUsesInjectedRepos(t *testing.T) {
	rels := []*gh.RepositoryRelease{
		{TagName: gh.Ptr("v9.0.0")},
		{TagName: gh.Ptr("v9.1.0")},
	}
	fake := &fakeReposForTest{releases: rels}
	ghh := GithubHelper{Client: nil, Repos: fake}

	data, err := ghh.FetchAllReleases("ftool", FetchOptions{Force: true, Limit: 0, RepoConf: RepoConfDef{}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data.Releases == nil || len(data.Releases) != len(rels) {
		t.Fatalf("expected %d releases, got %d", len(rels), len(data.Releases))
	}
}

func TestDownloadRelease_Success(t *testing.T) {
	td := t.TempDir()
	vrsPath := filepath.Join(td, "versions")
	tool := "dltool"
	version := "v1.2.3"

	osAlias := strings.ToLower(runtime.GOOS)
	archAlias := strings.ToLower(runtime.GOARCH)
	assetName := tool + "-" + osAlias + "-" + archAlias

	// prepare fake release with matching asset
	rel := &gh.RepositoryRelease{
		TagName: gh.Ptr(version),
		Assets:  []*gh.ReleaseAsset{{Name: gh.Ptr(assetName), ID: gh.Ptr(int64(123))}},
	}
	fake := &fakeReposForTest{releases: []*gh.RepositoryRelease{rel}}
	ghh := GithubHelper{Client: nil, Repos: fake}

	// call DownloadRelease
	if err := ghh.DownloadRelease(tool, version, vrsPath, RepoConfDef{Org: "o", Repo: "r"}); err != nil {
		t.Fatalf("DownloadRelease returned error: %v", err)
	}

	// check file exists and content
	dest := filepath.Join(vrsPath, tool, tool+"-"+version)
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(b) != "ok" {
		t.Fatalf("unexpected file content: %s", string(b))
	}

	// check executable bit set
	st, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if st.Mode().Perm()&0111 == 0 {
		t.Fatalf("expected executable bit set on %s", dest)
	}
}

func TestDownloadRelease_AssetNotFound(t *testing.T) {
	td := t.TempDir()
	vrsPath := filepath.Join(td, "versions")
	tool := "dltool"
	version := "v9.9.9"

	// create a release with non-matching asset name
	rel := &gh.RepositoryRelease{
		TagName: gh.Ptr(version),
		Assets:  []*gh.ReleaseAsset{{Name: gh.Ptr("other-asset"), ID: gh.Ptr(int64(1))}},
	}
	fake := &fakeReposForTest{releases: []*gh.RepositoryRelease{rel}}
	ghh := GithubHelper{Client: nil, Repos: fake}

	err := ghh.DownloadRelease(tool, version, vrsPath, RepoConfDef{Org: "o", Repo: "r"})
	if err == nil {
		t.Fatalf("expected error when asset not found")
	}
	if err != errReleaseNotFound {
		t.Fatalf("expected errReleaseNotFound, got: %v", err)
	}
}
