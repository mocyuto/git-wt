package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/template"
	"github.com/mocyuto/zgt/internal/zcontext"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Export environment variables for assigned ports",
	Long: `Generate shell export commands for the ports assigned to the current worktree.
Usage: eval "$(zgt env)"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = state.LoadState()

		if config.ConfigError != nil {
			return config.ConfigError
		}
		ports, found := state.GetCurrentWorktreePorts()
		if !found {
			// If not found, check if we are in the main git root.
			// If so, automatically assign index 0.
			gitRoot, err := gitroot.GetGitRoot()
			if err == nil {
				absGitRoot := state.NormalizePath(gitRoot)
				mainRoot, err := gitroot.GetMainProjectRoot()
				if err == nil {
					absMainRoot := state.NormalizePath(mainRoot)
					projectName := filepath.Base(absMainRoot)
					for name, basePort := range config.AppConfig.Ports {
						state.AssignPortIndex(projectName, absMainRoot, name, basePort)
					}
					// Also assign for the current git root if it's different (e.g. linked worktree)
					if absGitRoot != absMainRoot {
						for name, basePort := range config.AppConfig.Ports {
							state.AssignPortIndex(projectName, absGitRoot, name, basePort)
						}
					}
					_ = state.SaveState()
					ports, found = state.GetCurrentWorktreePorts()
				}
			}
		}

		if !found {
			return logger.Errorf("current directory is not a managed worktree")
		}

		if len(config.AppConfig.Ports) > 0 {
			for name, basePort := range config.AppConfig.Ports {
				idx, ok := ports[name]
				if !ok {
					continue
				}
				envName := strings.ToUpper(name) + "_PORT"
				port := basePort + idx
				fmt.Printf("export %s=%d\n", envName, port)
			}
		}

		if len(config.AppConfig.Env) > 0 {
			cwd, _ := os.Getwd()
			ctx := zcontext.New(cwd, "")

			replacedEnv := template.ReplaceMap(config.AppConfig.Env, ctx)
			for k, v := range replacedEnv {
				fmt.Printf("export %s=%q\n", k, v)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
