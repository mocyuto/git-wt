package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "WINDOW\tPANE\tSTATUS\tPROCESS")

	for _, win := range windows {
		status, err := tmux.GetWindowStatus(win.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get status for window %s: %v\n", win.ID, err)
			continue
		}

		// Filter only windows created by zgt (heuristic: name starts with [)
		// Or just show all if we want to be more permissive, but the user asked for zgt-monitor-tmux
		if !strings.HasPrefix(win.Name, "[") {
			// continue // Uncomment if we only want zgt windows
		}

		for i, pane := range status.Panes {
			windowName := win.Name
			if i > 0 {
				windowName = "" // Only show window name for the first pane
			}
			statusStr := "Waiting"
			if pane.IsRunning {
				statusStr = "Running"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", windowName, pane.ID, statusStr, pane.Running)
		}
	}
	w.Flush()
	return nil
}
