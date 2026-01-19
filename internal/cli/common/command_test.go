package common

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/github"
)

// helper to find a subcommand by use string
func findSubcmd(root *cobra.Command, use string) *cobra.Command {
	for _, c := range root.Commands() {
		if c.Use == use {
			return c
		}
	}
	return nil
}

func TestInitCommand_RegistersCommonSubcommands(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	repo := github.RepoConfDef{}
	InitCommand(root, "toolname", repo)

	// ensure list, list-remote and use exist
	if findSubcmd(root, "list") == nil {
		t.Fatalf("list subcommand not registered")
	}
	if findSubcmd(root, "list-remote") == nil {
		t.Fatalf("list-remote subcommand not registered")
	}
	if findSubcmd(root, "use <version>") == nil {
		t.Fatalf("use subcommand not registered")
	}
	if findSubcmd(root, "install <version>") == nil {
		t.Fatalf("install subcommand not registered")
	}
}
