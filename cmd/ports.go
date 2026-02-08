package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "List all port assignments",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = LoadState()
		CleanupState()
		_ = SaveState()

		if len(AppState.Worktrees) == 0 {
			fmt.Println("No port assignments found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "INDEX\tWORKTREE PATH")

		// Sort by index
		type assignment struct {
			path string
			idx  int
		}
		var assignments []assignment
		for p, i := range AppState.Worktrees {
			assignments = append(assignments, assignment{p, i})
		}
		sort.Slice(assignments, func(i, j int) bool {
			return assignments[i].idx < assignments[j].idx
		})

		for _, a := range assignments {
			fmt.Fprintf(w, "%d\t%s\n", a.idx, a.path)
		}
		w.Flush()

		if len(AppConfig.Ports) > 0 {
			fmt.Println("\nConfigured Base Ports:")
			for name, port := range AppConfig.Ports {
				fmt.Printf("  - %s: %d\n", name, port)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(portsCmd)
}
