package tools

import (
	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/cli/common"
	"github.com/stepbeta/vrsr/internal/github"
)

var (
	kindCmd = &cobra.Command{
		Use:   "kind",
		Short: "Manage kind versions",
		Long:  "A tool to easily install and use multiple versions of kind.",
	}
)

func NewKindCommand() *cobra.Command {
	return kindCmd
}

func init() {
	common.InitCommand(kindCmd, "kind", github.RepoConfDef{
		Org:  "kubernetes-sigs",
		Repo: "kind",
	})
}
