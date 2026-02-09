package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/git"
	"github.com/mocyuto/git-wt/internal/logger"
	"github.com/mocyuto/git-wt/internal/state"
	"github.com/mocyuto/git-wt/internal/template"
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
			return logger.Errorf("current directory is not a managed worktree index")
		}

		// 1. Export ports
		if len(config.AppConfig.Ports) > 0 {
			cmd.Println("# Port Assignments")
			for name, basePort := range config.AppConfig.Ports {
				envName := strings.ToUpper(name) + "_PORT"
				port := basePort + idx
				cmd.Printf("export %s=%d\n", envName, port)
			}
		}

		// 2. Export env from config with placeholder replacement
		if len(config.AppConfig.Env) > 0 {
			cmd.Println("# Custom Environment Variables")
			cwd, _ := os.Getwd()
			gitRoot, _ := git.GetGitRoot()
			branch, _ := git.GetCurrentBranch()

			ctx := template.Context{
				Path:   cwd,
				Branch: branch,
				Repo:   filepath.Base(gitRoot),
			}

			replacedEnv := template.ReplaceMap(config.AppConfig.Env, ctx)
			for k, v := range replacedEnv {
				cmd.Printf("export %s=%q\n", k, v)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
