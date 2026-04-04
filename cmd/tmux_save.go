package cmd

import (
	"fmt"

	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/spf13/cobra"
)

var tmuxSaveCmd = &cobra.Command{
	Use:   "save [session-name]",
	Short: "Save current tmux windows to state",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		currentSession, _ := tmux.GetCurrentSessionName()
		sessionName := currentSession
		if len(args) > 0 {
			sessionName = args[0]
		}

		if sessionName == "" {
			return fmt.Errorf("could not determine session name. please specify name or run inside tmux")
		}

		// Use the specified session to list windows
		windows, err := tmux.ListSessionWindows(sessionName)
		if err != nil {
			return fmt.Errorf("failed to list windows for session %s: %v", sessionName, err)
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
		state.AppState.TmuxSessions[sessionName] = saveState
		if err := state.SaveState(); err != nil {
			return err
		}

		logger.Success("Saved tmux session '%s' with %d windows", sessionName, len(windows))
		return nil
	},
}

func init() {
	tmuxCmd.AddCommand(tmuxSaveCmd)
}
