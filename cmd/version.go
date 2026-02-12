package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of git-wt",
	Long:  `All software has versions. This is git-wt's.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("git-wt version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
