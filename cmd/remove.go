package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var forceDelete bool

var removeCmd = &cobra.Command{
	Use:     "remove <branch>",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree by branch name",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchOrPath := args[0]

		path, err := findPathByBranch(branchOrPath)
		if err != nil {
			fmt.Printf("Notice: Branch '%s' not found in worktree list, trying as path...\n", branchOrPath)
			path = branchOrPath
		}

		fmt.Printf("--- Removing worktree at %s ---\n", path)
		if err := RemoveWorktree(path, forceDelete); err != nil {
			return fmt.Errorf("error removing worktree: %v", err)
		}
		fmt.Println("--- Done! ---")
		return nil
	},
}

func findPathByBranch(branch string) (string, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	var currentPath string
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			// branch name is refs/heads/BRANCHNAME
			fullBranch := strings.TrimPrefix(line, "branch ")
			if strings.HasSuffix(fullBranch, "/"+branch) {
				return currentPath, nil
			}
		}
	}
	return "", fmt.Errorf("branch not found")
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
