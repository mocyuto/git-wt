package cmd

import (
	"fmt"
	"strings"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/state"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Export environment variables for assigned ports",
	Long: `Generate shell export commands for the ports assigned to the current worktree.
Usage: eval $(git-wt env)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = state.LoadState()
		idx, found := state.GetCurrentWorktreePortIndex()
		if !found {
			return fmt.Errorf("current directory is not a managed worktree index")
		}

		if len(config.AppConfig.Ports) == 0 {
			cmd.Println("# No ports configured in git-wt.config.yml")
			return nil
		}

		for name, basePort := range config.AppConfig.Ports {
			envName := strings.ToUpper(name) + "_PORT"
			port := basePort + idx
			cmd.Printf("export %s=%d\n", envName, port)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
