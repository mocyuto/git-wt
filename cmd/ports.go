package cmd

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/state"
	"github.com/spf13/cobra"
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "List all port assignments",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = state.LoadState()
		state.CleanupState()
		_ = state.SaveState()

		if len(state.AppState.Worktrees) == 0 {
			cmd.Println("No port assignments found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "INDEX\tWORKTREE PATH")

		// Sort by index
		type assignment struct {
			path string
			idx  int
		}
		var assignments []assignment
		for p, i := range state.AppState.Worktrees {
			assignments = append(assignments, assignment{p, i})
		}
		sort.Slice(assignments, func(i, j int) bool {
			return assignments[i].idx < assignments[j].idx
		})

		for _, a := range assignments {
			fmt.Fprintf(w, "%d\t%s\n", a.idx, a.path)
		}
		w.Flush()

		if len(config.AppConfig.Ports) > 0 {
			cmd.Println("\nConfigured Base Ports:")
			for name, port := range config.AppConfig.Ports {
				cmd.Printf("  - %s: %d\n", name, port)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(portsCmd)
}
