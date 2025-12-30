package common

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/github"
	"github.com/stepbeta/vrsr/internal/utils"
)

// newGithubInstallCommand creates a new 'install' command for the specified tool.
// Installation is done via GitHub releases.
func newGithubInstallCommand(tool string, repoConf github.RepoConfDef) *cobra.Command {
	return &cobra.Command{
		Use:   "install <version>",
		Short: fmt.Sprintf("Download and install %s for the current OS/ARCH", tool),
		Long: fmt.Sprintf("Download the %s binary for the current OS/ARCH at the specified version.\n\n"+
			"This binary will be saved into the path specified by the \"bin-path\" flag. It will be named \"%s-$version\".\n\n"+
			"Make sure to check the \"use <version>\" command after installing a new version", tool, tool),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installGithub(cmd, args[0], tool, repoConf)
		},
	}
}

// installGithub downloads and installs the specified version of the tool from GitHub releases
func installGithub(cmd *cobra.Command, vrs, tool string, repoConf github.RepoConfDef) error {
	if utils.IsToolInUse(tool, vrs) {
		cmd.Printf("%s version %s is already in use. Nothing to do\n", tool, vrs)
		return nil
	}
	if utils.IsToolInstalled(tool, vrs) {
		cmd.Printf("%s version %s is already installed.\n", tool, vrs)
		cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
		return nil
	}

	vrsPath := viper.GetString("vrs-path")
	ghc := github.New(nil)
	if err := ghc.DownloadRelease(tool, vrs, vrsPath, repoConf); err != nil {
		return err
	}

	cmd.Printf("%s version %s successfully installed\n", tool, vrs)
	cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
	return nil
}

// newDownloadInstallCommand creates a new 'install' command for the specified tool
func newDownloadInstallCommand(tool string, repoConf github.RepoConfDef) *cobra.Command {
	return &cobra.Command{
		Use:   "install <version>",
		Short: fmt.Sprintf("Download and install %s for the current OS/ARCH", tool),
		Long: fmt.Sprintf("Download the %s binary for the current OS/ARCH at the specified version.\n\n"+
			"This binary will be saved into the path specified by the \"bin-path\" flag. It will be named \"%s-$version\".\n\n"+
			"Make sure to check the \"use <version>\" command after installing a new version", tool, tool),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installDownload(cmd, args[0], tool, repoConf)
		},
	}
}

// installDownload downloads and installs the specified version of the tool from GitHub releases
func installDownload(cmd *cobra.Command, vrs, tool string, repoConf github.RepoConfDef) error {
	if utils.IsToolInUse(tool, vrs) {
		cmd.Printf("%s version %s is already in use. Nothing to do\n", tool, vrs)
		return nil
	}
	if utils.IsToolInstalled(tool, vrs) {
		cmd.Printf("%s version %s is already installed.\n", tool, vrs)
		cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
		return nil
	}
	vrsPath := viper.GetString("vrs-path")
	if err := utils.DownloadBinary(repoConf.DownloadURL, tool, vrs, vrsPath, repoConf.Zipped); err != nil {
		return err
	}
	cmd.Printf("%s version %s successfully installed\n", tool, vrs)
	cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
	return nil
}
