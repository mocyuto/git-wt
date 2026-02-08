package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	forceDelete bool
	keepBranch  bool
)

var removeCmd = &cobra.Command{
	Use:     "remove <branch>",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree by branch name",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchOrPath := args[0]
		var path, branch string

		// Try to resolve branch and path
		p, b, err := resolveWorktreeInfo(branchOrPath)
		if err != nil {
			fmt.Printf("Notice: Could not resolve worktree for '%s' in list, trying as path...\n", branchOrPath)
			path = branchOrPath
			// If it was a path, we might still want to find its branch for deletion
			_, branch, _ = resolveWorktreeInfo(path)
		} else {
			path = p
			branch = b
		}

		fmt.Printf("--- Removing worktree at %s ---\n", path)
		if err := RemoveWorktree(path, forceDelete); err != nil {
			return fmt.Errorf("error removing worktree: %v", err)
		}

		// Delete branch if not kept
		if !keepBranch && branch != "" {
			fmt.Printf("--- Deleting branch %s ---\n", branch)
			if err := DeleteBranch(branch); err != nil {
				fmt.Printf("Warning: failed to delete branch %s: %v\n", branch, err)
			}
		}

		// Run removal hooks
		gitRoot, _ := GetGitRoot()
		RunHooks("rm", HookContext{
			Path:   path,
			Branch: branch,
			Repo:   filepath.Base(gitRoot),
		})

		// Release port index
		absPath, _ := filepath.Abs(path)
		_ = LoadState()
		ReleasePortIndex(absPath)
		CleanupState()
		_ = SaveState()

		fmt.Println("--- Done! ---")
		return nil
	},
}

func resolveWorktreeInfo(search string) (path, branch string, err error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(string(out), "\n")
	var currentPath string
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			fullBranch := strings.TrimPrefix(line, "branch ")
			branchName := strings.TrimPrefix(fullBranch, "refs/heads/")

			// Match by branch name or path
			if branchName == search || currentPath == search || strings.HasSuffix(fullBranch, "/"+search) {
				return currentPath, branchName, nil
			}
		}
	}
	return "", "", fmt.Errorf("not found")
}

func init() {
	removeCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "force removal even if worktree is dirty")
	removeCmd.Flags().BoolVarP(&keepBranch, "keep-branch", "k", false, "do not delete the associated branch")
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

func DeleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
