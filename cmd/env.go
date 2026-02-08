package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Export environment variables for assigned ports",
	Long: `Generate shell export commands for the ports assigned to the current worktree.
Usage: eval $(git-wt env)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = LoadState()
		idx, found := GetCurrentWorktreePortIndex()
		if !found {
			return fmt.Errorf("current directory is not a managed worktree index")
		}

		if len(AppConfig.Ports) == 0 {
			fmt.Println("# No ports configured in git-wt.config.yml")
			return nil
		}

		for name, basePort := range AppConfig.Ports {
			envName := strings.ToUpper(name) + "_PORT"
			port := basePort + idx
			fmt.Printf("export %s=%d\n", envName, port)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
