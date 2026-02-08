package cmd

import (
	"os"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "git-wt",
	Short: "Create git worktree and copy ignored files",
	Long: `git-wt is a CLI tool that extends 'git worktree add' by automatically
copying ignored configuration files (like .env) from the main tree to the new worktree.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(config.InitConfig)

	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "config file (default is $HOME/.config/git-wt/config.yaml)")
}
