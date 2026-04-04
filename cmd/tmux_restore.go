package cmd

import (
	"fmt"
	"os/exec"

	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/spf13/cobra"
)

var tmuxRestoreCmd = &cobra.Command{
	Use:   "restore [session-name]",
	Short: "Restore tmux windows from state",
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

		if err := state.LoadState(); err != nil {
			return err
		}

		session, ok := state.AppState.TmuxSessions[sessionName]
		if !ok {
			return fmt.Errorf("no saved session found for: %s", sessionName)
		}

		for _, w := range session.Windows {
			// Check if a window with this name already exists
			_, exists, _ := tmux.GetWindowIDByName(w.Name)
			if exists {
				logger.Info("Window '%s' already exists, skipping", w.Name)
				continue
			}

			if w.IsZgt {
				logger.Info("Restoring zgt window: %s (worktree: %s)", w.Name, w.ZgtWorktree)
				// Call tmuxOpenCmd.RunE
				if err := tmuxOpenCmd.RunE(cmd, []string{w.ZgtWorktree}); err != nil {
					logger.Warn("Failed to restore zgt window %s: %v", w.Name, err)
				}
			} else {
				logger.Info("Restoring window: %s (CWD: %s)", w.Name, w.CWD)
				// tmux new-window -n [name] -c [cwd]
				newWindowCmd := exec.Command("tmux", "new-window", "-n", w.Name, "-c", w.CWD)
				if err := newWindowCmd.Run(); err != nil {
					logger.Warn("Failed to restore window %s: %v", w.Name, err)
				}
			}
		}

		logger.Success("Restored windows for session '%s'", sessionName)
		return nil
	},
}

func init() {
	tmuxCmd.AddCommand(tmuxRestoreCmd)
}
