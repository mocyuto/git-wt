package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var forceDelete bool

var removeCmd = &cobra.Command{
	Use:     "remove <path>",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := args[0]
		fmt.Printf("--- Removing worktree at %s ---\n", targetPath)
		if err := RemoveWorktree(targetPath, forceDelete); err != nil {
			return fmt.Errorf("error removing worktree: %v", err)
		}
		fmt.Println("--- Done! ---")
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "force removal even if worktree is dirty")
	rootCmd.AddCommand(removeCmd)
}

func RemoveWorktree(path string, force bool) error {
	cmdArgs := []string{"worktree", "remove", path}
	if force {
		cmdArgs = append(cmdArgs, "--force")
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
