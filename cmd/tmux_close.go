package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/mocyuto/zgt/internal/zcontext"
	"github.com/spf13/cobra"
)

var tmuxCloseCmd = &cobra.Command{
	Use:     "close <branch>",
	Aliases: []string{"kill"},
	Short:   "Gracefully close tmux window for a worktree",
	Args:    cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		completions, err := git.GetWorktreeCompletions()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		search := args[0]
		path, branch, err := git.ResolveWorktreeInfo(search)
		if err != nil {
			return fmt.Errorf("failed to resolve worktree info for %s: %v", search, err)
		}

		// Resolve profile from state so the profile-aware window_name and
		// keep_open semantics match `zgt add` / `zgt tmux open` / `zgt rm`.
		absPath := state.NormalizePath(path)
		mainRoot, _ := gitroot.GetMainProjectRoot()
		projectName := filepath.Base(mainRoot)
		_ = state.LoadState()
		profile := ""
		if proj, ok := state.AppState.Projects[projectName]; ok {
			if wt, ok := proj.Worktrees[absPath]; ok {
				profile = wt.Profile
			}
		}

		ctx := zcontext.NewWithProfile(path, branch, profile)
		fmt.Printf("Gracefully closing tmux window for: %s\n", branch)
		return tmux.CloseWindow(ctx, true)
	},
}

func init() {
	tmuxCmd.AddCommand(tmuxCloseCmd)
}
