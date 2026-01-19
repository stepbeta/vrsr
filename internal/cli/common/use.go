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
	installOnUse   bool
	errVrsNotFound = errors.New("version not found")
)

// newUseCommand creates a new 'use' command for the specified tool
func newUseCommand(tool string) *cobra.Command {
	useCmd := &cobra.Command{
		Use:   "use <version>",
		Short: fmt.Sprintf("Set the specified %s version as the active one", tool),
		Long:  fmt.Sprintf(`Create a symlink to the specified version with the name "%s".\n\nMake sure the "bin-path" is included in the $PATH variable.`, tool),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return use(cmd, args[0], tool)
		},
	}
	// Bind flags to Viper keys so config file / env / flags work together.
	useCmd.Flags().BoolVarP(&installOnUse, "install", "i", false, "Install the version if not yet present (best effort)")
	if err := viper.BindPFlag(fmt.Sprintf("%s.use.install", tool), useCmd.Flags().Lookup("install")); err != nil {
		useCmd.PrintErr(err)
		panic(err)
	}
	return useCmd
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
		installOnUse = viper.GetBool(tool + ".use.install")
		if !installOnUse {
			cmd.Printf("Error: specified version is not installed. Please install it first using `vrsr %s install <version>`", tool)
			return errVrsNotFound
		}
		pCmd := cmd.Parent()
		if pCmd == nil {
			return fmt.Errorf("internal error: use command has no parent")
		}
		iCmd, _, err := pCmd.Find([]string{"install"})
		if err != nil || iCmd.Name() != "install" {
			return fmt.Errorf("could not find sibling 'install' command")
		}
		if err := iCmd.RunE(iCmd, []string{vrs, "true"}); err != nil {
			cmd.Println("Error executing install:", err)
			cmd.Println("Skipping action")
			return err
		}
		// here we should have installed the version, we assume it succeeded
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
