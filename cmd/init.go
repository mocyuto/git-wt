package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize git-wt configuration",
	Long:  `Create a default git-wt.config.yml in the project root directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		root, err := git.GetMainProjectRoot()
		if err != nil {
			logger.Warn("Not in a git repository. Creating config in current directory.")
			root = "."
		}
		configPath := filepath.Join(root, "git-wt.config.yml")
		altConfigPath := filepath.Join(root, "git-wt.config.yaml")

		// Check if the file already exists (both .yml and .yaml)
		var existingPath string
		if _, err := os.Stat(configPath); err == nil {
			existingPath = configPath
		} else if _, err := os.Stat(altConfigPath); err == nil {
			existingPath = altConfigPath
		}

		if existingPath != "" {
			logger.Warn("%s already exists. Skipping creation.", existingPath)
		} else {
			defaultConfig := `ports:
  api: 3000
env:
  COMPOSE_PROJECT_NAME: "{{.Repo}}-{{.Branch}}"
hooks:
  add:
    - "tmux new-window -d -n [{{.Repo}}]{{.Branch}} -c {{.Path}} 'claude; exec $SHELL'"
`
			err := os.WriteFile(configPath, []byte(defaultConfig), 0644)
			if err != nil {
				logger.Error("failed to create config file: %v", err)
				return
			}
			logger.Success("Created %s", configPath)
		}

		fmt.Println("\nUsage of 'add' command:")
		fmt.Printf("  git-wt add <branch>          # Create worktree at ../%s-{branch}\n", filepath.Base(root))
		fmt.Println("  git-wt add <path> <branch>   # Create worktree at specified path")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
