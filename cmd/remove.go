package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/mocyuto/git-wt/internal/git"
	"github.com/mocyuto/git-wt/internal/hook"
	"github.com/mocyuto/git-wt/internal/state"
	"github.com/mocyuto/git-wt/internal/template"
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
		p, b, err := git.ResolveWorktreeInfo(branchOrPath)
		if err != nil {
			fmt.Printf("Notice: Could not resolve worktree for '%s' in list, trying as path...\n", branchOrPath)
			path = branchOrPath
			// If it was a path, we might still want to find its branch for deletion
			_, branch, _ = git.ResolveWorktreeInfo(path)
		} else {
			path = p
			branch = b
		}

		fmt.Printf("--- Removing worktree at %s ---\n", path)
		if err := git.RemoveWorktree(path, forceDelete); err != nil {
			return fmt.Errorf("error removing worktree: %v", err)
		}

		// Delete branch if not kept
		if !keepBranch && branch != "" {
			fmt.Printf("--- Deleting branch %s ---\n", branch)
			if err := git.DeleteBranch(branch); err != nil {
				fmt.Printf("Warning: failed to delete branch %s: %v\n", branch, err)
			}
		}

		// Run removal hooks
		gitRoot, _ := git.GetGitRoot()
		hook.RunHooks("rm", template.Context{
			Path:   path,
			Branch: branch,
			Repo:   filepath.Base(gitRoot),
		})

		// Release port index
		absPath, _ := filepath.Abs(path)
		_ = state.LoadState()
		state.ReleasePortIndex(absPath)
		state.CleanupState()
		_ = state.SaveState()

		fmt.Println("--- Done! ---")
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "force removal even if worktree is dirty")
	removeCmd.Flags().BoolVarP(&keepBranch, "keep-branch", "k", false, "do not delete the associated branch")
	rootCmd.AddCommand(removeCmd)
}
