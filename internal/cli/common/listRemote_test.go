package common

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/github"
)

func TestListRemoteFlags(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	repo := github.RepoConfDef{}
	InitCommand(root, "toolname", repo)

	lr := findSubcmd(root, "list-remote")
	if lr == nil {
		t.Fatalf("list-remote subcommand not registered")
	}

	if lr.Flags().Lookup("devel") == nil {
		t.Fatalf("expected 'devel' flag on list-remote")
	}
	if lr.Flags().Lookup("limit") == nil {
		t.Fatalf("expected 'limit' flag on list-remote")
	}
	if lr.Flags().Lookup("force") == nil {
		t.Fatalf("expected 'force' flag on list-remote")
	}

	// default values
	if lr.Flags().Lookup("devel").Value.String() != "false" {
		t.Fatalf("expected default devel=false")
	}
	if lr.Flags().Lookup("limit").Value.String() != "0" {
		t.Fatalf("expected default limit=0")
	}
	if lr.Flags().Lookup("force").Value.String() != "false" {
		t.Fatalf("expected default force=false")
	}
}
