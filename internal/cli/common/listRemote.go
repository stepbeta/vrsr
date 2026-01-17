package common

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stepbeta/vrsr/internal/github"
	"github.com/stepbeta/vrsr/internal/utils"
)

var (
	includeDevel bool
	limit        int
	forceRefresh bool
)

// newGithubListRemoteCommand creates a new 'list-remote' command for the specified tool
func newGithubListRemoteCommand(tool string, repoConf github.RepoConfDef) *cobra.Command {
	listRemoteCmd := &cobra.Command{
		Use:   "list-remote",
		Short: fmt.Sprintf("List all remote %s versions from GitHub (sorted by semver)", tool),
		Long: fmt.Sprintf("Lists all the remote %s versions available as GitHub releases (sorted by semver).\n\n"+
			"In the list the versions currently installed are marked with a '+' symbol, while the version currently in use is marked with a '*' symbol.\n\n"+
			"Note: By default pre-release versions (alpha, beta, rc) are hidden. Use '--devel' to include them", tool),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRemoteGithub(cmd, tool, repoConf)
		},
	}
	// Bind flags to Viper keys so config file / env / flags work together.
	listRemoteCmd.Flags().BoolVar(&includeDevel, "devel", false, "Include pre-release versions (alpha, beta, rc)")
	if err := viper.BindPFlag(fmt.Sprintf("%s.list-remote.devel", tool), listRemoteCmd.Flags().Lookup("devel")); err != nil {
		listRemoteCmd.PrintErr(err)
		panic(err)
	}
	listRemoteCmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of versions displayed")
	if err := viper.BindPFlag(fmt.Sprintf("%s.list-remote.limit", tool), listRemoteCmd.Flags().Lookup("limit")); err != nil {
		listRemoteCmd.PrintErr(err)
		panic(err)
	}
	listRemoteCmd.Flags().BoolVarP(&forceRefresh, "force", "f", false, "Force refresh of remote versions cache")
	if err := viper.BindPFlag(fmt.Sprintf("%s.list-remote.force", tool), listRemoteCmd.Flags().Lookup("force")); err != nil {
		listRemoteCmd.PrintErr(err)
		panic(err)
	}
	return listRemoteCmd
}

// listRemoteGithub lists all remote versions of the specified tool available as GitHub releases (sorted by semver)
func listRemoteGithub(cmd *cobra.Command, tool string, repoConf github.RepoConfDef) error {
	includeDevel = viper.GetBool(tool + ".list-remote.devel")
	limit = viper.GetInt(tool + ".list-remote.limit")
	forceRefresh = viper.GetBool(tool + ".list-remote.force")
	ghc := github.New(nil)
	releasesData, err := ghc.FetchAllReleases(tool, github.FetchOptions{
		IncludeDevel: includeDevel,
		Limit:        limit,
		Force:        forceRefresh,
		RepoConf:     repoConf,
	})
	if err != nil {
		return err
	}
	versions := utils.SemverFromReleases(releasesData.Releases, includeDevel)

	// ignore errors here, it's not important
	currentVersion := ""
	binPath := viper.GetString("bin-path")
	if binPath != "" {
		currentVersion, err = utils.GetVrsInUse(binPath, tool)
		if err != nil {
			cmd.Println("Error getting current version:", err)
		}
	}

	// ignore errors here, it's not important
	var localVersions []*semver.Version
	vrsPath := viper.GetString("vrs-path")
	if vrsPath != "" {
		localVersions, err = utils.ListInstalledVersions(vrsPath, tool)
		if err != nil {
			cmd.Println("Error listing available binaries:", err)
		}
	}
	cmd.Println("Available versions to download:")
	if limit > 0 && len(versions) > limit {
		versions = versions[len(versions)-limit:]
	}
	for _, v := range versions {
		vrs := v.Original()
		if vrs == currentVersion {
			vrs += " *"
		} else {
			for _, lv := range localVersions {
				if lv.Equal(v) {
					vrs += " +"
					break
				}
			}
		}
		cmd.Println(vrs)
	}

	if !includeDevel {
		cmd.Println("\nNote: Pre-release versions (alpha, beta, rc) are hidden. Use '--devel' to include them.")
	}
	if !forceRefresh && time.Since(releasesData.Timestamp) > 5*time.Minute {
		cmd.Printf("\nNote: The results shown above were cached %s. You can use the '-f' flag to force a refresh of the list.\n", humanize.Time(releasesData.Timestamp))
	}
	return nil
}
