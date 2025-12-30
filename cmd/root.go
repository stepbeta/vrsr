package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/cli/tools"
	"github.com/stepbeta/vrsr/internal/utils"
)

// rootCmd represents the base command when called without any subcommands
var (
	cfgFile string
	// override at build time using `go build -ldflags "-X github.com/stepbeta/vrsr/cmd.Version=x.y.z"`
	Version = "0.0.1"
	rootCmd = &cobra.Command{
		Use:   "vrsr",
		Short: "(Almost) Universal tools versions manager",
		Long:  `A tool to easily install and use multiple versions of several tools.`,
		// PersistentPreRunE is called after flags are parsed but before the
		// command's RunE function is called.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// global flags
	defaultBinPath, err := utils.GetDefaultBinPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error determining default bin path:", err)
		os.Exit(1)
	}
	rootCmd.PersistentFlags().StringP("bin-path", "b", defaultBinPath, "Absolute path to folder storing in-use tools binaries")
	defaultVrsPath, err := utils.GetDefaultVrsPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error determining default vrs path:", err)
		os.Exit(1)
	}
	rootCmd.PersistentFlags().StringP("vrs-path", "d", defaultVrsPath, "Absolute path to folder storing downloaded tools binary versions")

	// commands
	rootCmd.AddCommand(tools.NewKindCommand())
	rootCmd.AddCommand(tools.NewTalosctlCommand())
	rootCmd.AddCommand(tools.NewKubectlCommand())
	rootCmd.AddCommand(tools.NewHelmCommand())
}

// initializeConfig sets up Viper to read in config files and environment variables
func initializeConfig(cmd *cobra.Command) error {
	// 1. Set up Viper to use environment variables.
	viper.SetEnvPrefix("VRSR")
	// Allow for nested keys in environment variables (e.g. `VRSR_BIN_PATH`)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// 2. Handle the configuration file.
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for a config file in default locations.
		home, err := os.UserHomeDir()
		// Only panic if we can't get the home directory.
		cobra.CheckErr(err)

		// Search for a config file with the name "config" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath(home + "/.vrsr")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 3. Read the configuration file.
	// If a config file is found, read it in. We use a robust error check
	// to ignore "file not found" errors, but panic on any other error.
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist.
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	// 4. Bind Cobra flags to Viper.
	// This is the magic that makes the flag values available through Viper.
	// It binds the full flag set of the command passed in.
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return err
	}

	// This is an optional but useful step to debug your config.
	// fmt.Println("Configuration initialized. Using config file:", viper.ConfigFileUsed())
	return nil
}
