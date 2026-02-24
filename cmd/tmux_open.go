package cmd

import (
	"fmt"

	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/mocyuto/zgt/internal/zcontext"
	"github.com/spf13/cobra"
)

var tmuxOpenCmd = &cobra.Command{
	Use:   "open [worktree-name]",
	Short: "Open or activate tmux window for a worktree",
	Args:  cobra.ExactArgs(1),
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
		windowName := tmux.GetWindowName(ctx)

		windowID, exists, err := tmux.GetWindowIDByName(windowName)
		if err != nil {
			return err
		}

		if exists {
			fmt.Printf("Activating existing tmux window: %s\n", windowName)
			return tmux.ActivateWindow(windowID)
		}

		fmt.Printf("Creating new tmux window: %s\n", windowName)
		return tmux.Setup(ctx)
	},
}

func init() {
	tmuxCmd.AddCommand(tmuxOpenCmd)
}
