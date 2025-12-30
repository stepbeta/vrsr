package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/github"
)

func TestUseCommand_ArgsValidation(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	repo := github.RepoConfDef{}
	InitCommand(root, "toolname", repo)

	useCmd := findSubcmd(root, "use <version>")
	if useCmd == nil {
		t.Fatalf("use subcommand not registered")
	}

	// no args -> should return error
	if err := useCmd.Args(useCmd, []string{}); err == nil {
		t.Fatalf("expected error when calling use with no args")
	}
	// one arg -> should be ok
	if err := useCmd.Args(useCmd, []string{"1.2.3"}); err != nil {
		t.Fatalf("expected no error when calling use with one arg: %v", err)
	}
	// two args -> should return error
	if err := useCmd.Args(useCmd, []string{"1.2.3", "quack"}); err == nil {
		t.Fatalf("expected error when calling use with two args: %v", err)
	}
}

func TestUse_CreatesSymlink(t *testing.T) {
	td := t.TempDir()
	vrsPath := filepath.Join(td, "versions")
	binPath := filepath.Join(td, "bin")

	tool := "sample"
	version := "0.0.1"
	// create vrs binary
	if err := os.MkdirAll(filepath.Join(vrsPath, tool), 0o755); err != nil {
		t.Fatalf("failed to create vrs dir: %v", err)
	}
	filePath := filepath.Join(vrsPath, tool, tool+"-"+version)
	if err := os.WriteFile(filePath, []byte("x"), 0o755); err != nil {
		t.Fatalf("failed to create vrs file: %v", err)
	}

	viper.Set("vrs-path", vrsPath)
	viper.Set("bin-path", binPath)

	if err := use(&cobra.Command{}, version, tool); err != nil {
		t.Fatalf("use failed: %v", err)
	}

	// check symlink exists and points to correct file
	link := filepath.Join(binPath, tool)
	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatalf("expected symlink at %s, eval error: %v", link, err)
	}
	if target != filePath {
		t.Fatalf("symlink target mismatch: expected %s got %s", filePath, target)
	}
}
