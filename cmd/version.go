package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of zgt",
	Long:  `All software has versions. This is zgt's.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("zgt version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
