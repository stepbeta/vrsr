package tools

import (
	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/cli/common"
	"github.com/stepbeta/vrsr/internal/github"
)

var (
	helmCmd = &cobra.Command{
		Use:   "helm",
		Short: "Manage helm versions",
		Long:  "A tool to easily install and use multiple versions of helm.",
	}
)

func NewHelmCommand() *cobra.Command {
	return helmCmd
}

func init() {
	common.InitCommand(helmCmd, "helm", github.RepoConfDef{
		Org:    "helm",
		Repo:   "helm",
		Zipped: true,
		// Example: "https://get.helm.sh/helm-v4.0.0-linux-amd64.tar.gz"
		DownloadURL: "https://get.helm.sh/helm-%s-%s-%s",
	})
}
