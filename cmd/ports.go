package cmd

import (
	"fmt"
	"sort"
	"strings"
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

		// Get sorted port names from config
		var portNames []string
		for name := range config.AppConfig.Ports {
			portNames = append(portNames, name)
		}
		sort.Strings(portNames)

		// Header
		fmt.Fprint(w, "INDEX\tWORKTREE PATH")
		for _, name := range portNames {
			fmt.Fprintf(w, "\t%s", strings.ToUpper(name))
		}
		fmt.Fprintln(w)

		// Sort assignments by index
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
			fmt.Fprintf(w, "%d\t%s", a.idx, a.path)
			for _, name := range portNames {
				basePort := config.AppConfig.Ports[name]
				fmt.Fprintf(w, "\t%d", basePort+a.idx)
			}
			fmt.Fprintln(w)
		}
		w.Flush()

		if len(config.AppConfig.Ports) > 0 {
			cmd.Println("\nBase Ports Configuration:")
			for _, name := range portNames {
				cmd.Printf("  - %s: %d\n", name, config.AppConfig.Ports[name])
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(portsCmd)
}
