package cmd

import (
	"fmt"

	"github.com/mocyuto/zgt/internal/tmux"
	"github.com/spf13/cobra"
)

var tmuxLsCmd = &cobra.Command{
	Use:     "ls",
	Short:   "List all tmux windows and their status",
	Aliases: []string{"list"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return ListTmuxWindows()
	},
}

func init() {
	tmuxCmd.AddCommand(tmuxLsCmd)
}

func ListTmuxWindows() error {
	windows, err := tmux.ListWindows()
	if err != nil {
		return err
	}

	if len(windows) == 0 {
		fmt.Println("No tmux windows found.")
		return nil
	}

	// Group windows by session
	sessionWindows := make(map[string][]tmux.WindowStatus)
	var sessions []string
	for _, win := range windows {
		if _, ok := sessionWindows[win.SessionName]; !ok {
			sessions = append(sessions, win.SessionName)
		}
		sessionWindows[win.SessionName] = append(sessionWindows[win.SessionName], win)
	}

	currentSession, _ := tmux.GetCurrentSessionName()

	for i, session := range sessions {
		sessionDisplay := session
		if session == currentSession {
			sessionDisplay = fmt.Sprintf("%s (current)", session)
		}
		fmt.Printf("%s\n", sessionDisplay)
		wins := sessionWindows[session]
		for j, win := range wins {
			winPrefix := "├─"
			if j == len(wins)-1 {
				winPrefix = "└─"
			}
			fmt.Printf("%s %s (%s)\n", winPrefix, win.Name, win.ID)

			status, err := tmux.GetWindowStatus(win.ID)
			if err != nil {
				fmt.Printf("   (Error getting status: %v)\n", err)
				continue
			}

			for k, pane := range status.Panes {
				panePrefix := "│  ├─"
				if j == len(wins)-1 {
					panePrefix = "   ├─"
				}
				if k == len(status.Panes)-1 {
					if j == len(wins)-1 {
						panePrefix = "   └─"
					} else {
						panePrefix = "│  └─"
					}
				}

				statusStr := "Waiting"
				if pane.IsRunning {
					statusStr = "Running"
				}
				fmt.Printf("%s %s %s: %s\n", panePrefix, pane.ID, statusStr, pane.Running)
			}
		}
		if i < len(sessions)-1 {
			fmt.Println()
		}
	}

	return nil
}
