package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize zgt configuration",
	Long:  `Create a default zgt.config.yml in the project root directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		root, err := gitroot.GetMainProjectRoot()
		if err != nil {
			logger.Warn("Not in a git repository. Creating config in current directory.")
			root = "."
		}

		existingPath := config.GetLocalConfigPath(root)

		if existingPath != "" {
			logger.Warn("%s already exists. Skipping creation.", existingPath)
		} else {
			configPath := filepath.Join(root, config.DefaultLocalConfigName)
			defaultConfig := `ports:
  api: 3000
env:
  COMPOSE_PROJECT_NAME: "{{.Repo}}-{{.Branch}}"
tmux:
  enabled: true
  panes:
    - id: main
      commands: ["yarn"]
    - id: dev
      target: main
      split: horizontal
      size: 50%
      commands: ["yarn dev"]
`
			err := os.WriteFile(configPath, []byte(defaultConfig), 0644)
			if err != nil {
				logger.Error("failed to create config file: %v", err)
				return
			}
			logger.Success("Created %s", configPath)
		}

		fmt.Println("\nUsage of 'add' command:")
		fmt.Printf("  zgt add <branch>          # Create worktree at ../%s-{branch}\n", filepath.Base(root))
		fmt.Println("  zgt add <path> <branch>   # Create worktree at specified path")

		// Add config files to .gitignore if not present
		ignorePath := filepath.Join(root, ".gitignore")
		data, err := os.ReadFile(ignorePath)
		if err != nil && !os.IsNotExist(err) {
			logger.Warn("failed to read .gitignore: %v", err)
			return
		}

		content := string(data)
		configsToIgnore := config.LocalConfigNames
		var toAdd []string
		for _, cfg := range configsToIgnore {
			if !contains(content, cfg) {
				toAdd = append(toAdd, cfg)
			}
		}

		if len(toAdd) > 0 {
			f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Warn("failed to open .gitignore: %v", err)
				return
			}
			defer f.Close()

			if content != "" && !strings.HasSuffix(content, "\n") {
				if _, err := f.WriteString("\n"); err != nil {
					logger.Warn("failed to write to .gitignore: %v", err)
					return
				}
			}

			for _, cfg := range toAdd {
				if _, err := f.WriteString(cfg + "\n"); err != nil {
					logger.Warn("failed to write to .gitignore: %v", err)
					return
				}
			}
			logger.Success("Added %v to .gitignore", toAdd)
		}
	},
}

func contains(content, target string) bool {
	lines := strings.SplitSeq(content, "\n")
	for line := range lines {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(initCmd)
}
