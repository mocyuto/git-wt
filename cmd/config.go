package cmd

import (
	"fmt"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display the merged configuration",
	Long:  "Display the merged configuration from global, project, and command-line sources in YAML format.",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := yaml.Marshal(config.AppConfig)
		if err != nil {
			return fmt.Errorf("error marshaling config: %v", err)
		}

		fmt.Println("--- Merged Configuration ---")
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
