package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/stepbeta/vrsr/internal/utils"
)

var docsPath = "./docs"

// docsCmd represents the docs command
var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "generate vrsr documentation",
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.EnsurePathExists(docsPath); err != nil {
			cmd.Printf("Error creating docs directory: %v\n", err)
			return
		}
		if err := doc.GenMarkdownTree(rootCmd, docsPath); err != nil {
			cmd.Printf("Error generating docs: %v\n", err)
			return
		}
		cmd.Printf("Documentation generated in %s\n", docsPath)
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
