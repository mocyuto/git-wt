package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/git"
	"github.com/mocyuto/git-wt/internal/logger"
	"github.com/mocyuto/git-wt/internal/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configCheck bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display the merged configuration",
	Long:  "Display the merged configuration from global, project, and command-line sources in YAML format.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configCheck {
			if config.ConfigError != nil {
				return config.ConfigError
			}
			logger.Success("Configuration is valid.")
			return nil
		}

		// Prepare context for placeholder replacement
		cwd, _ := os.Getwd()
		gitRoot, _ := git.GetGitRoot()
		branch, _ := git.GetCurrentBranch()

		ctx := template.Context{
			Path:   cwd,
			Branch: branch,
			Repo:   filepath.Base(gitRoot),
		}

		replacedConfig := template.ReplaceConfig(config.AppConfig, ctx)

		data, err := yaml.Marshal(replacedConfig)
		if err != nil {
			return logger.Errorf("error marshaling config: %v", err)
		}

		fmt.Println("--- Merged Configuration (Placeholders Replaced) ---")
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().BoolVar(&configCheck, "check", false, "check configuration for errors")
}
