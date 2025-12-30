package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
)

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
		return extractSpecificFile(resp.Body, internalArchivePath, destPath, resp.ContentLength)
	}

	// Direct binary logic (your original code)
	tmpFile, err := os.CreateTemp(finalPath, tool+"-download-*")
	if err != nil {
		return err
	}
	defer func() {
		// Clean up if we don't rename
		if err := os.Remove(tmpFile.Name()); err != nil {
			fmt.Printf("warning: failed to remove temp file: %v\n", err)
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

	return os.Chmod(destPath, 0755)
}

// extractSpecificFile extracts a specific file from a gzip-compressed tar archive.
func extractSpecificFile(gzipStream io.Reader, internalPath, destPath string, size int64) error {
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

		// 4. Check if the current entry matches our dynamic path
		// We use filepath.ToSlash to ensure cross-platform path consistency
		if header.Name == internalPath || filepath.Clean(header.Name) == internalPath {
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create destination: %w", err)
			}
			defer func() {
				if err = outFile.Close(); err != nil {
					fmt.Printf("warning: failed to close output file: %v\n", err)
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
