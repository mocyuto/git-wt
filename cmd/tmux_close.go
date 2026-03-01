package cmd

import (
	"fmt"

	"github.com/mocyuto/zgt/internal/git"
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

		ctx := zcontext.New(path, branch)
		fmt.Printf("Gracefully closing tmux window for: %s\n", branch)
		return tmux.CloseWindow(ctx, true)
	},
}

func init() {
	tmuxCmd.AddCommand(tmuxCloseCmd)
}
