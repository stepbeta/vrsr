package github

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
		if osAlias != "windows" && !repo.Zipped && len(strings.Split(lname, ".")) > 1 {
			// non-windows binaries should not have an extension
			continue
		}
		if osAlias != "windows" && repo.Zipped && !strings.HasSuffix(lname, ".tar.gz") {
			// non-windows zipped binaries must have .tar.gz suffix
			continue
		}
		asset = a
		break
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

	// write to temp file then move (safer) or extract if zipped
	finalPath := filepath.Join(vrsPath, tool)
	err = utils.EnsurePathExists(finalPath)
	if err != nil {
		return fmt.Errorf("error ensuring vrs path exists: %w", err)
	}

	// If the repo provides a zipped archive, extract the specific binary
	if repo.Zipped {
		destPath := filepath.Join(finalPath, tool+"-"+version)
		if err := utils.ExtractSpecificFile(rc, tool, destPath, -1); err != nil {
			return fmt.Errorf("failed to extract file from archive: %w", err)
		}
		return nil
	}

	// Non-zipped: save binary to temp file then move into place
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

	// verify checksum if available
	if err := verifyDownloadChecksum(gh, ctx, rel, asset.GetName(), destPath); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	return nil
}

// isChecksumAsset returns true if the asset name looks like a checksum file.
func isChecksumAsset(name string) bool {
	ln := strings.ToLower(name)
	return strings.HasSuffix(ln, ".sha256") ||
		strings.HasSuffix(ln, ".sha256sum") ||
		strings.HasSuffix(ln, ".sha256.sig") ||
		strings.Contains(ln, "checksums") ||
		ln == "sha256sums" ||
		ln == "sha256sums.txt" ||
		ln == "sha256sum.txt"
}

// parseChecksums parses a checksum file in standard sha256sum format.
// Each line is: "<hex-hash>  <filename>" or "<hex-hash> *<filename>".
// Returns a map of filename → hash.
func parseChecksums(r io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(r)
	checksums := make(map[string]string)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		var hash, name string
		if len(fields) >= 2 {
			hash = fields[0]
			name = strings.TrimLeft(fields[1], "*")
		} else if len(fields) == 1 {
			hash = fields[0]
		}
		if len(hash) != 64 || name == "" {
			continue
		}
		checksums[name] = hash
	}
	return checksums, scanner.Err()
}

// verifyDownloadChecksum searches release assets for a checksum file, downloads
// it, and verifies the downloaded binary against the matching hash.
// Returns nil if no checksum file is found (best-effort).
func verifyDownloadChecksum(gh *GithubHelper, ctx context.Context, rel *github.RepositoryRelease, assetName, destPath string) error {
	// Find a checksum asset
	var checksumAsset *github.ReleaseAsset
	for _, a := range rel.Assets {
		if a == nil {
			continue
		}
		if isChecksumAsset(a.GetName()) || a.GetName() == assetName+".sha256" {
			checksumAsset = a
			break
		}
	}

	// If the explicit .sha256 asset exists but wasn't caught above, try matching
	if checksumAsset == nil {
		for _, a := range rel.Assets {
			if a == nil {
				continue
			}
			if a.GetName() == assetName+".sha256" {
				checksumAsset = a
				break
			}
		}
	}

	if checksumAsset == nil {
		fmt.Fprintf(os.Stderr, "warning: no checksum file found for %s, skipping verification\n", assetName)
		return nil
	}

	// We don't have the repo owner/name here, derive from any known context.
	// For now use the asset download URL directly.
	u := checksumAsset.GetBrowserDownloadURL()
	if u == "" {
		fmt.Fprintf(os.Stderr, "warning: checksum file %s has no download URL\n", checksumAsset.GetName())
		return nil
	}
	resp, err := http.Get(u)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to download checksum file %s: %v\n", checksumAsset.GetName(), err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "warning: bad status downloading checksum file %s: %s\n", checksumAsset.GetName(), resp.Status)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to read checksum file: %v\n", err)
		return nil
	}

	// Try parsing as standard sha256sum format first
	checksums, parseErr := parseChecksums(bytes.NewReader(body))
	if parseErr == nil && len(checksums) > 0 {
		hash, ok := checksums[assetName]
		if !ok {
			// Try without the archive extension
			hash, ok = checksums[strings.TrimSuffix(assetName, ".tar.gz")]
		}
		if ok {
			return verifyFileSHA256(destPath, hash)
		}
	}

	// Fallback: treat the entire file as just the hex hash
	hash := strings.TrimSpace(string(body))
	hash = strings.Fields(hash)[0]
	if len(hash) == 64 {
		return verifyFileSHA256(destPath, hash)
	}

	fmt.Fprintf(os.Stderr, "warning: could not find hash for %s in checksum file %s\n", assetName, checksumAsset.GetName())
	return nil
}

// verifyFileSHA256 computes the SHA256 of the file at path and compares it to expectedHex.
func verifyFileSHA256(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum verification: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to compute SHA256: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expectedHex) {
		return fmt.Errorf("SHA256 mismatch for %s: expected %s, got %s", path, expectedHex, got)
	}
	fmt.Fprintf(os.Stderr, "info: checksum verification passed for %s\n", path)
	return nil
}
