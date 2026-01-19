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
	installType := InstallGitHubCmd
	if repoConf.DownloadURL != "" {
		installType = InstallDownloadCmd
	}
	cmd.AddCommand(newInstallCommand(tool, repoConf, installType))
	// use
	cmd.AddCommand(newUseCommand(tool))
}
