package tools

import (
	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/cli/common"
	"github.com/stepbeta/vrsr/internal/github"
)

var (
	hubbleCmd = &cobra.Command{
		Use:   "hubble",
		Short: "Manage hubble versions",
		Long:  "A tool to easily install and use multiple versions of hubble.",
	}
)

func NewHubbleCommand() *cobra.Command {
	return hubbleCmd
}

func init() {
	common.InitCommand(hubbleCmd, "hubble", github.RepoConfDef{
		Org:    "cilium",
		Repo:   "hubble",
		Zipped: true,
	})
}
