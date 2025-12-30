package utils

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v78/github"
	"github.com/spf13/viper"
)

// GetCachePath returns the path to the releases cache file.
func GetCachePath(tool string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	binPath := filepath.Join(homeDir, ".vrsr", tool+"-releases.json")
	return binPath, nil
}

// GetDefaultBinPath returns the default bin path for in-use tool binary.
func GetDefaultBinPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	binPath := filepath.Join(homeDir, ".vrsr", "bin")
	return binPath, nil
}

// GetDefaultVrsPath returns the default bin path for downloaded tool binaries.
func GetDefaultVrsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	binPath := filepath.Join(homeDir, ".vrsr", "versions")
	return binPath, nil
}

// EnsurePathExists ensures that the given path exists, creating it if necessary.
func EnsurePathExists(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// ListInstalledVersions lists all installed tool versions in the given vrsPath.
func ListInstalledVersions(vrsPath, tool string) ([]*semver.Version, error) {
	files, err := os.ReadDir(filepath.Join(vrsPath, tool))
	if err != nil {
		if os.IsNotExist(err) {
			// vrsPath does not exist, return empty list
			return []*semver.Version{}, nil
		}
		return nil, err
	}
	versions := make([]*semver.Version, 0)
	for _, f := range files {
		fileName := f.Name()
		if f.IsDir() || !strings.HasPrefix(fileName, tool) {
			// skip directories and non-tool files
			continue
		}
		// by convention the file name is tool-VERSION
		fv := strings.Split(fileName, "-")
		if fv == nil || len(fv) != 2 {
			// skip unexpected file names
			continue
		}
		v, err := semver.NewVersion(fv[1])
		if err == nil {
			versions = append(versions, v)
		}
	}
	sort.Sort(semver.Collection(versions))

	return versions, nil
}

// GetVrsInUse returns the version of tool currently in use from the given binPath.
func GetVrsInUse(binPath, tool string) (string, error) {
	linkPath, err := filepath.EvalSymlinks(filepath.Join(binPath, tool))
	if os.IsNotExist(err) || linkPath == "" {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	baseName := filepath.Base(linkPath)
	parts := strings.Split(baseName, "-")
	if len(parts) == 2 {
		return parts[1], nil
	}
	return "", nil
}

// SemverFromReleases extracts semver.Version objects from GitHub releases.
func SemverFromReleases(releases []*github.RepositoryRelease, includeDevel bool) []*semver.Version {
	var versions []*semver.Version
	for _, r := range releases {
		v, err := semver.NewVersion(*r.TagName)
		if err == nil {
			if !includeDevel && v.Prerelease() != "" {
				continue
			}
			versions = append(versions, v)
		}
	}

	sort.Sort(semver.Collection(versions))
	return versions
}

// IsToolInstalled checks if the specified version of the tool is installed in the vrsPath.
func IsToolInstalled(tool, vrs string) bool {
	vrsPath := viper.GetString("vrs-path")
	fileNameWithPath := filepath.Join(vrsPath, tool, tool+"-"+vrs)
	if _, err := os.Stat(fileNameWithPath); err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		return false
	}
	// err on the side of caution and say not installed
	return false
}

// IsToolInUse checks if the specified version of the tool is currently in use.
func IsToolInUse(tool, vrs string) bool {
	var currentVersion string
	var err error
	binPath := viper.GetString("bin-path")
	if binPath != "" {
		currentVersion, err = GetVrsInUse(binPath, tool)
		if err != nil {
			return false
		}
	}
	if currentVersion == vrs {
		return true
	}
	return false
}
