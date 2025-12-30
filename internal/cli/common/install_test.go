package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/github"
)

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

	// calling installGithub should return early (tool already in use)
	if err := installGithub(&cobra.Command{}, "1.2.3", "mytool", github.RepoConfDef{}); err != nil {
		t.Fatalf("installGithub expected nil error when tool in use, got: %v", err)
	}

	// remove symlink to test IsToolInstalled early return
	if err := os.Remove(symlink); err != nil {
		t.Fatalf("failed to remove symlink: %v", err)
	}

	// calling installDownload should return early (tool already installed)
	if err := installDownload(&cobra.Command{}, "1.2.3", "mytool", github.RepoConfDef{}); err != nil {
		t.Fatalf("installDownload expected nil error when tool installed, got: %v", err)
	}
}
