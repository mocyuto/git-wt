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

		if len(state.AppState.Projects) == 0 {
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
		fmt.Fprint(w, "PROJECT\tWORKTREE PATH")
		for _, name := range portNames {
			fmt.Fprintf(w, "\t%s", strings.ToUpper(name))
		}
		fmt.Fprintln(w)

		// Collect and sort assignments by project then path
		type assignment struct {
			projectName string
			path        string
			ports       map[string]int
		}
		var assignments []assignment
		for pn, proj := range state.AppState.Projects {
			for p, wt := range proj.Worktrees {
				assignments = append(assignments, assignment{pn, p, wt.Ports})
			}
		}
		sort.Slice(assignments, func(i, j int) bool {
			if assignments[i].projectName != assignments[j].projectName {
				return assignments[i].projectName < assignments[j].projectName
			}
			return assignments[i].path < assignments[j].path
		})

		for _, a := range assignments {
			fmt.Fprintf(w, "%s\t%s", a.projectName, a.path)
			for _, name := range portNames {
				if idx, ok := a.ports[name]; ok {
					basePort := config.AppConfig.Ports[name]
					fmt.Fprintf(w, "\t%d", basePort+idx)
				} else {
					fmt.Fprint(w, "\t-")
				}
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
