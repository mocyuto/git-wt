package cmd

import (
	"fmt"
	"os"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/spf13/cobra"
)

var Version = ""

var rootCmd = &cobra.Command{
	Use:           "zgt",
	Short:         "Create git worktree and copy ignored files",
	Version:       Version,
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `zgt is a CLI tool that extends 'git worktree add' by automatically
copying ignored configuration files (like .env) from the main tree to the new worktree.`,
}

func Execute(version string) {
	Version = version
	rootCmd.Version = Version
	if err := rootCmd.Execute(); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		config.InitConfig()
		if logger.Verbose {
			config.PrintConfig()
		}
	})

	desc := "config file"
	if globalPath, err := config.GetGlobalConfigPath(); err == nil {
		desc = fmt.Sprintf("config file (default is %s)", globalPath)
	}
	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", desc)
	rootCmd.PersistentFlags().BoolVarP(&logger.Verbose, "verbose", "v", false, "show detailed output")
}
