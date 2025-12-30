package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/v78/github"
)

// SaveToCache saves release data to cache file
func SaveToCache(tool string, allReleases []*github.RepositoryRelease) {
	releasesData, err := json.Marshal(ReleasesData{
		Timestamp: time.Now().UTC(),
		Releases:  allReleases,
	})
	if err != nil {
		fmt.Println("Failed to marshal release data to json:", err)
		return
	}
	cachePath, err := GetCachePath(tool)
	if err != nil {
		fmt.Println("Failed to retrieve cache path:", err)
		return
	}
	err = EnsurePathExists(filepath.Dir(cachePath))
	if err != nil {
		fmt.Println("Failed to create cache dir:", err)
		return
	}
	err = os.WriteFile(cachePath, []byte(releasesData), 0644)
	if err != nil {
		fmt.Println("Failed to save release data to cache:", err)
		return
	}
}

// ReadFromCache reads cached release data from file
func ReadFromCache(tool string, limit int) (ReleasesData, error) {
	var err error
	cachePath, err := GetCachePath(tool)
	if err != nil {
		fmt.Println("Failed to retrieve cache path:", err)
		return ReleasesData{}, err
	}
	content, err := os.ReadFile(cachePath)
	if os.IsNotExist(err) {
		// cache file not found
		return ReleasesData{}, nil
	}
	if err != nil {
		fmt.Println("Failed to read cache data from file:", err)
		return ReleasesData{}, err
	}
	var cacheData ReleasesData
	err = json.Unmarshal(content, &cacheData)
	if err != nil {
		fmt.Println("Failed to unmarshal cache data:", err)
		return ReleasesData{}, err
	}
	// apply limit if found
	if limit > 0 && len(cacheData.Releases) > limit {
		return ReleasesData{
			Timestamp: cacheData.Timestamp,
			Releases:  cacheData.Releases[0:limit],
		}, nil
	}
	return cacheData, nil
}

type ReleasesData struct {
	Timestamp time.Time                   `json:"timestamp"`
	Releases  []*github.RepositoryRelease `json:"releases"`
}
