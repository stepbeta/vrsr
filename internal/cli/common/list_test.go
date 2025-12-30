package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestList_NoVersionsInstalled(t *testing.T) {
	td := t.TempDir()
	viper.Set("vrs-path", td)

	cmd := &cobra.Command{}
	var sb strings.Builder
	cmd.SetOut(&sb)

	if err := list(cmd, "mytool"); err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "No mytool versions installed.") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestList_MarksInUseVersion(t *testing.T) {
	td := t.TempDir()
	vrsPath := filepath.Join(td, "versions")
	tool := "marktool"
	toolDir := filepath.Join(vrsPath, tool)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}

	// create multiple versions
	for _, v := range []string{"0.1.0", "1.0.0", "2.0.0"} {
		p := filepath.Join(toolDir, tool+"-"+v)
		if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
			t.Fatalf("failed to write file %s: %v", p, err)
		}
	}

	// create bin and symlink to 1.0.0 as current
	binPath := filepath.Join(td, "bin")
	if err := os.MkdirAll(binPath, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	linkTarget := filepath.Join(toolDir, tool+"-"+"1.0.0")
	if err := os.Symlink(linkTarget, filepath.Join(binPath, tool)); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	viper.Set("vrs-path", vrsPath)
	viper.Set("bin-path", binPath)

	cmd := &cobra.Command{}
	var sb strings.Builder
	cmd.SetOut(&sb)

	if err := list(cmd, tool); err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "Available marktool versions:") {
		t.Fatalf("unexpected output header: %s", out)
	}
	// ensure the in-use version is marked with an asterisk
	if !strings.Contains(out, "1.0.0 *") {
		t.Fatalf("expected in-use version 1.0.0 to be marked with '*', got: %s", out)
	}
	// ensure other versions are present
	if !strings.Contains(out, "0.1.0") || !strings.Contains(out, "2.0.0") {
		t.Fatalf("expected other versions to be listed, got: %s", out)
	}
}
