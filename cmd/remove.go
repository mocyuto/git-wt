package cmd

import (
	"path/filepath"

	"strings"

	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/hook"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/template"
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
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		wts, err := git.GetWorktrees()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, wt := range wts {
			if wt.Branch != "" {
				// strip refs/heads/
				branch := strings.TrimPrefix(wt.Branch, "refs/heads/")
				completions = append(completions, branch)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		branchOrPath := args[0]
		var path, branch string

		// Try to resolve branch and path
		p, b, err := git.ResolveWorktreeInfo(branchOrPath)
		if err != nil {
			logger.Info("Notice: Could not resolve worktree for '%s' in list, trying as path...", branchOrPath)
			path = branchOrPath
			// If it was a path, we might still want to find its branch for deletion
			_, branch, _ = git.ResolveWorktreeInfo(path)
		} else {
			path = p
			branch = b
		}

		logger.Info("--- Removing worktree at %s ---", path)
		if err := git.RemoveWorktree(path, forceDelete); err != nil {
			return logger.Errorf("error removing worktree: %v", err)
		}

		// Delete branch if not kept
		if !keepBranch && branch != "" {
			logger.Info("--- Deleting branch %s ---", branch)
			if err := git.DeleteBranch(branch); err != nil {
				logger.Warn("failed to delete branch %s: %v", branch, err)
			}
		}

		// Run removal hooks
		mainRoot, _ := gitroot.GetMainProjectRoot()
		currentRoot, _ := gitroot.GetGitRoot()
		currentBranch, _ := git.GetCurrentBranch()
		hook.RunHooks("rm", template.Context{
			Path:          path,
			Branch:        branch,
			Repo:          filepath.Base(mainRoot),
			CurrentDir:    filepath.Base(currentRoot),
			TargetBranch:  branch,
			CurrentBranch: currentBranch,
		})

		// Release port index
		absPath := state.NormalizePath(path)
		projectName := filepath.Base(mainRoot)
		_ = state.LoadState()
		state.ReleasePortIndex(projectName, absPath)
		state.CleanupState()
		_ = state.SaveState()

		logger.Success("--- Done!---")
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "force removal even if worktree is dirty")
	removeCmd.Flags().BoolVarP(&keepBranch, "keep-branch", "k", false, "do not delete the associated branch")
	rootCmd.AddCommand(removeCmd)
}
