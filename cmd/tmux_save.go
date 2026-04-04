package cmd

import (
	"fmt"

	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/spf13/cobra"
)

var sessionNameFlag string

var tmuxSaveCmd = &cobra.Command{
	Use:   "save [save-name]",
	Short: "Save current tmux windows to state",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Identify the source tmux session
		sourceSession := sessionNameFlag
		if sourceSession == "" {
			var err error
			sourceSession, err = tmux.GetCurrentSessionName()
			if err != nil || sourceSession == "" {
				return fmt.Errorf("could not determine current tmux session. please specify with --session or run inside tmux")
			}
		}

		// Identify the save name (identifier in state)
		saveName := sourceSession
		if len(args) > 0 {
			saveName = args[0]
		}

		// Use the source session to list windows
		windows, err := tmux.ListSessionWindows(sourceSession)
		if err != nil {
			return fmt.Errorf("failed to list windows for session %s: %v", sourceSession, err)
		}

		saveState := state.TmuxSessionState{}
		for _, w := range windows {
			managed, _ := tmux.GetWindowProperty(w.ID, "#{@zgt-managed}")
			worktree, _ := tmux.GetWindowProperty(w.ID, "#{@zgt-worktree}")

			saveState.Windows = append(saveState.Windows, state.TmuxWindowSaveState{
				Name:        w.Name,
				CWD:         w.CWD,
				IsZgt:       managed == "1",
				ZgtWorktree: worktree,
			})
		}

		if err := state.LoadState(); err != nil {
			return err
		}
		state.AppState.TmuxSessions[saveName] = saveState
		if err := state.SaveState(); err != nil {
			return err
		}

		logger.Success("Saved tmux session '%s' as '%s' with %d windows", sourceSession, saveName, len(windows))
		return nil
	},
}

func init() {
	tmuxSaveCmd.Flags().StringVarP(&sessionNameFlag, "session", "s", "", "source tmux session name to save")
	tmuxCmd.AddCommand(tmuxSaveCmd)
}
