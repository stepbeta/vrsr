package common

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/github"
	"github.com/stepbeta/vrsr/internal/utils"
)

var useOnInstall bool

type InstallCmdType int

const (
	InstallGitHubCmd InstallCmdType = iota
	InstallDownloadCmd
)

// newGithubInstallCommand creates a new 'install' command for the specified tool.
func newInstallCommand(tool string, repoConf github.RepoConfDef, installType InstallCmdType) *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "install <version>",
		Short: fmt.Sprintf("Download and install %s for the current OS/ARCH", tool),
		Long: fmt.Sprintf("Download the %s binary for the current OS/ARCH at the specified version.\n\n"+
			"This binary will be saved into the path specified by the \"bin-path\" flag. It will be named \"%s-$version\".\n\n"+
			"Make sure to check the \"use <version>\" command after installing a new version", tool, tool),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skipMsg := len(args) > 1 && args[1] == "true"
			return install(cmd, args[0], tool, repoConf, installType, skipMsg)
		},
	}
	// Bind flags to Viper keys so config file / env / flags work together.
	installCmd.Flags().BoolVarP(&useOnInstall, "use", "u", false, "Immediately use the version once installed (best effort)")
	if err := viper.BindPFlag(fmt.Sprintf("%s.install.use", tool), installCmd.Flags().Lookup("use")); err != nil {
		installCmd.PrintErr(err)
		panic(err)
	}
	return installCmd
}

// install downloads and installs the specified version of the tool from GitHub releases
func install(cmd *cobra.Command, vrs, tool string, repoConf github.RepoConfDef, installType InstallCmdType, skipMsg bool) error {
	if utils.IsToolInUse(tool, vrs) {
		cmd.Printf("%s version %s is already installed and in use. Nothing to do\n", tool, vrs)
		return nil
	}
	useOnInstall = viper.GetBool(tool + ".install.use")
	if utils.IsToolInstalled(tool, vrs) {
		cmd.Printf("%s version %s is already installed.\n", tool, vrs)
		if !useOnInstall {
			if !skipMsg {
				cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
			}
			return nil
		}
		if err := useOnInstallFn(cmd, vrs, tool); err != nil {
			return err
		}
	}

	vrsPath := viper.GetString("vrs-path")
	// depending on the install type we use the appropriate install method
	switch installType {
	case InstallGitHubCmd:
		ghc := github.New(nil)
		if err := ghc.DownloadRelease(tool, vrs, vrsPath, repoConf); err != nil {
			return err
		}
	case InstallDownloadCmd:
		if err := utils.DownloadBinary(repoConf.DownloadURL, tool, vrs, vrsPath, repoConf.Zipped); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown install type")
	}
	cmd.Printf("%s version %s successfully installed\n", tool, vrs)

	if !useOnInstall {
		if !skipMsg {
			cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
		}
		return nil
	}
	return useOnInstallFn(cmd, vrs, tool)
}

// useOnInstallFn attempts to use the installed version immediately
func useOnInstallFn(cmd *cobra.Command, vrs, tool string) error {
	pCmd := cmd.Parent()
	if pCmd == nil {
		// this is a sign of something wrong, so we return an error
		return fmt.Errorf("internal error: install command has no parent")
	}
	uCmd, _, err := pCmd.Find([]string{"use"})
	if err != nil || uCmd.Name() != "use" {
		cmd.Println("could not find sibling 'use' command")
		cmd.Println("Skipping action")
		cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
		// this is a best effort, if there's an error we just return
		return nil
	}
	if err := uCmd.RunE(uCmd, []string{vrs}); err != nil {
		cmd.Println("Error executing use:", err)
		cmd.Println("Skipping action")
		cmd.Printf("To switch to that version run `vrsr %s use %s`\n", tool, vrs)
		// this is a best effort, if there's an error we just return
		return nil
	}
	return nil
}
