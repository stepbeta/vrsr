package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// VerifySHA256 computes the SHA256 of r and compares it to expectedHex.
func VerifySHA256(r io.Reader, expectedHex string) error {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return fmt.Errorf("failed to compute SHA256: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expectedHex) {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHex, got)
	}
	return nil
}

// tryVerifyDownload attempts to fetch a .sha256 file for the given URL and verify
// the downloaded file at destPath against it. It is best-effort: if no checksum
// file is found a warning is printed to stderr but no error is returned.
func tryVerifyDownload(url, destPath string) {
	checksumURL := url + ".sha256"
	resp, err := http.Get(checksumURL)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to read checksum file from %s: %v\n", checksumURL, err)
		return
	}

	// Parse the expected hash from the response body.
	// Two common formats:
	//   1. Just the hex hash on a line (e.g. kubectl)
	//   2. "<hash>  <filename>" standard sha256sum output
	hash := ""
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Format: <hash>  <filename> or <hash> *<filename>
			hash = fields[0]
		} else {
			// Format: just the hash
			hash = fields[0]
		}
		if len(hash) == 64 {
			break
		}
	}

	if len(hash) != 64 {
		fmt.Fprintf(os.Stderr, "warning: no valid SHA256 found in checksum file from %s\n", checksumURL)
		return
	}

	f, err := os.Open(destPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to open downloaded file for checksum verification: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()

	if err := VerifySHA256(f, hash); err != nil {
		fmt.Fprintf(os.Stderr, "error: checksum verification failed: %v\n", err)
		if rmErr := os.Remove(destPath); rmErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove file after failed verification: %v\n", rmErr)
		}
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "info: checksum verification passed for %s\n", destPath)
}

// DownloadBinary downloads a binary from the specified URL, handling both zipped and direct binaries.
func DownloadBinary(dlURL, tool, version, vrsPath string, zipped bool) error {
	osAlias := strings.ToLower(runtime.GOOS)
	archAlias := strings.ToLower(runtime.GOARCH)

	// 1. Append extension if zipped
	fullURL := fmt.Sprintf(dlURL, version, osAlias, archAlias)
	if zipped {
		fullURL += ".tar.gz"
	}

	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	finalPath := filepath.Join(vrsPath, tool)
	if err := EnsurePathExists(finalPath); err != nil {
		return err
	}
	destPath := filepath.Join(finalPath, tool+"-"+version)

	// 2. Branch logic based on zipped flag
	if zipped {
		// Construct the expected path inside the tar: "linux-amd64/toolname"
		internalArchivePath := fmt.Sprintf("%s-%s/%s", osAlias, archAlias, tool)
		if err := ExtractSpecificFile(resp.Body, internalArchivePath, destPath, resp.ContentLength); err != nil {
			return err
		}
		tryVerifyDownload(fullURL, destPath)
		return nil
	}

	// Direct binary logic
	tmpFile, err := os.CreateTemp(finalPath, tool+"-download-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmpFile.Name())
		}
	}()

	bar := progressbar.DefaultBytes(resp.ContentLength, "Downloading...")
	if _, err = io.Copy(io.MultiWriter(tmpFile, bar), resp.Body); err != nil {
		return err
	}
	if err = tmpFile.Close(); err != nil {
		return err
	}

	if err = os.Rename(tmpFile.Name(), destPath); err != nil {
		return err
	}

	if err = os.Chmod(destPath, 0755); err != nil {
		return err
	}
	tryVerifyDownload(fullURL, destPath)
	return nil
}

// ExtractSpecificFile extracts a specific file from a gzip-compressed tar archive.
func ExtractSpecificFile(gzipStream io.Reader, internalPath, destPath string, size int64) error {
	// 1. Setup progress bar for the download stream
	bar := progressbar.DefaultBytes(size, "Downloading & Extracting...")

	// 2. Initialize gzip reader
	uncompressedStream, err := gzip.NewReader(io.TeeReader(gzipStream, bar))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if err = uncompressedStream.Close(); err != nil {
			fmt.Printf("warning: failed to close gzip reader: %v\n", err)
		}
	}()

	tarReader := tar.NewReader(uncompressedStream)

	// 3. Iterate through archive entries
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %w", err)
		}

		// 4. Reject entries attempting directory traversal
		if strings.Contains(header.Name, "..") || filepath.IsAbs(header.Name) {
			fmt.Printf("warning: skipping unsafe archive entry: %s\n", header.Name)
			continue
		}

		// 5. Check if the current entry matches our dynamic path
		// We use filepath.ToSlash to ensure cross-platform path consistency
		if header.Name == internalPath || filepath.Clean(header.Name) == internalPath {
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create destination: %w", err)
			}
			defer func() {
				if cerr := outFile.Close(); cerr != nil {
					fmt.Printf("warning: failed to close output file: %v\n", cerr)
				}
			}()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("binary '%s' not found inside the archive", internalPath)
}
