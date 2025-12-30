package tools

import (
	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/cli/common"
	"github.com/stepbeta/vrsr/internal/github"
)

var (
	kubeCmd = &cobra.Command{
		Use:   "kubectl",
		Short: "Manage kubectl versions",
		Long:  "A tool to easily install and use multiple versions of kubectl.",
	}
)

func NewKubectlCommand() *cobra.Command {
	return kubeCmd
}

func init() {
	common.InitCommand(kubeCmd, "kubectl", github.RepoConfDef{
		Org:  "kubernetes",
		Repo: "kubernetes",
		// Example: "https://dl.k8s.io/release/v1.35.0/bin/linux/amd64/kubectl"
		DownloadURL: "https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
	})
}
