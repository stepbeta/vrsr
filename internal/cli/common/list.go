package common

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/utils"
)

// newGithubListCommand creates a new 'list' command for the specified tool
func newListCommand(tool string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List all installed %s versions", tool),
		Long:  fmt.Sprintf(`Lists all the %s versions that are currently installed on the system.`, tool),
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cmd, tool)
		},
	}
}

// list lists all installed versions of the specified tool
func list(cmd *cobra.Command, tool string) error {
	vrsPath := viper.GetString("vrs-path")
	versions, err := utils.ListInstalledVersions(vrsPath, tool)
	if err != nil {
		cmd.Println("Error listing available binaries:", err)
		return err
	}
	if len(versions) == 0 {
		cmd.Printf("No %s versions installed.\n", tool)
		return nil
	}

	// ignore errors here, it's not important
	currentVersion := ""
	binPath := viper.GetString("bin-path")
	if binPath != "" {
		currentVersion, err = utils.GetVrsInUse(binPath, tool)
		if err != nil {
			cmd.Println("Error getting current version:", err)
		}
	}

	cmd.Printf("Available %s versions:\n", tool)
	for _, v := range versions {
		vrs := v.Original()
		if vrs == currentVersion {
			vrs += " *"
		}
		cmd.Println(vrs)
	}
	return nil
}
