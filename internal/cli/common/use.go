package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/utils"
)

var (
	errVrsNotFound = errors.New("version not found")
)

// newUseCommand creates a new 'use' command for the specified tool
func newUseCommand(tool string) *cobra.Command {
	return &cobra.Command{
		Use:   "use <version>",
		Short: fmt.Sprintf("Set the specified %s version as the active one", tool),
		Long:  fmt.Sprintf(`Create a symlink to the specified version with the name "%s".\n\nMake sure the "bin-path" is included in the $PATH variable.`, tool),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return use(cmd, args[0], tool)
		},
	}
}

// use sets the specified version of the tool as the active one
func use(cmd *cobra.Command, vrs, tool string) error {
	binPath := viper.GetString("bin-path")
	err := utils.EnsurePathExists(binPath)
	if err != nil {
		cmd.Println("Error ensuring bin path exists:", err)
		return err
	}
	vrsPath := viper.GetString("vrs-path")
	fileName := filepath.Join(vrsPath, tool, tool+"-"+vrs)
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		cmd.Printf("Error: specified version is not installed. Please install it first using `vrsr %s install <version>`", tool)
		return errVrsNotFound
	}
	target := filepath.Join(binPath, tool)
	// Check if the symlink already exists
	if _, err := os.Lstat(target); err == nil {
		// Remove existing symlink or file
		if err := os.Remove(target); err != nil {
			cmd.Println("Error removing existing symlink:", err)
			return err
		}
	}
	// create new symlink
	if err := os.Symlink(fileName, target); err != nil {
		cmd.Println("Error creating symlink:", err)
		return err
	}

	cmd.Printf("Now using %s version %s\n", tool, vrs)
	return nil
}
