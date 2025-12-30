package common

import (
	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/github"
)

// InitCommand initializes the common commands for the specified tool
func InitCommand(cmd *cobra.Command, tool string, repoConf github.RepoConfDef) {
	// list
	cmd.AddCommand(newListCommand(tool))
	// list-remote
	cmd.AddCommand(newGithubListRemoteCommand(tool, repoConf))
	// install
	if repoConf.DownloadURL == "" {
		cmd.AddCommand(newGithubInstallCommand(tool, repoConf))
	} else {
		cmd.AddCommand(newDownloadInstallCommand(tool, repoConf))
	}
	// use
	cmd.AddCommand(newUseCommand(tool))
}
