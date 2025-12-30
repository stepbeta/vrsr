package tools

import (
	"github.com/spf13/cobra"
	"github.com/stepbeta/vrsr/internal/cli/common"
	"github.com/stepbeta/vrsr/internal/github"
)

var (
	talosCmd = &cobra.Command{
		Use:   "talosctl",
		Short: "Manage talosctl versions",
		Long:  "A tool to easily install and use multiple versions of talosctl.",
	}
)

func NewTalosctlCommand() *cobra.Command {
	return talosCmd
}

func init() {
	common.InitCommand(talosCmd, "talosctl", github.RepoConfDef{
		Org:  "siderolabs",
		Repo: "talos",
	})
}
