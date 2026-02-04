package cmd

import (
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "vrsr tool version",
	Long:  `Shows the current version of the vrsr tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("vrsr version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
