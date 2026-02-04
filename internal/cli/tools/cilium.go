package tools

import (
	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/cli/common"
	"github.com/stepbeta/vrsr/internal/github"
)

var (
	ciliumCmd = &cobra.Command{
		Use:   "cilium",
		Short: "Manage Cilium CLI versions",
		Long:  "A tool to easily install and use multiple versions of Cilium CLI.",
	}
)

func NewCiliumCommand() *cobra.Command {
	return ciliumCmd
}

func init() {
	common.InitCommand(ciliumCmd, "cilium", github.RepoConfDef{
		Org:    "cilium",
		Repo:   "cilium-cli",
		Zipped: true,
	})
}
