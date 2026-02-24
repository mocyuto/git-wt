package cmd

import (
	"github.com/spf13/cobra"
)

var tmuxCmd = &cobra.Command{
	Use:   "tmux",
	Short: "Tmux related commands",
}

func init() {
	rootCmd.AddCommand(tmuxCmd)
}
