package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/spf13/cobra"
)

var (
	showAllPorts bool
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "List all port assignments",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = state.LoadState()
		state.CleanupState()
		_ = state.SaveState()

		// Get current project name for filtering
		var currentProject string
		mainRoot, err := gitroot.GetMainProjectRoot()
		if err == nil {
			currentProject = filepath.Base(mainRoot)
		}

		// Collect all possible port names
		portNamesSet := make(map[string]bool)

		// Always include current project's config ports
		for name := range config.AppConfig.Ports {
			portNamesSet[name] = true
		}

		if showAllPorts {
			// Include everything from all projects in state
			for _, proj := range state.AppState.Projects {
				for _, wt := range proj.Worktrees {
					for name := range wt.Ports {
						portNamesSet[name] = true
					}
				}
			}
		} else if currentProject != "" {
			// Only include ports assigned in the current project
			if proj, ok := state.AppState.Projects[currentProject]; ok {
				for _, wt := range proj.Worktrees {
					for name := range wt.Ports {
						portNamesSet[name] = true
					}
				}
			}
		}

		var portNames []string
		for name := range portNamesSet {
			portNames = append(portNames, name)
		}
		sort.Strings(portNames)

		// Collect assignments and cache project base ports
		type assignment struct {
			projectName string
			path        string
			ports       map[string]int
			basePorts   map[string]int
		}
		var assignments []assignment
		projectBasePorts := make(map[string]map[string]int)

		for pn, proj := range state.AppState.Projects {
			if !showAllPorts && currentProject != "" && pn != currentProject {
				continue
			}

			// Try to load base ports for this project
			var basePorts map[string]int
			if pn == currentProject {
				basePorts = config.AppConfig.Ports
			} else {
				// Find first valid worktree path to get repo root
				for p := range proj.Worktrees {
					root, err := gitroot.GetMainProjectRootFromPath(p)
					if err == nil {
						bp, err := config.LoadPortsFromPath(root)
						if err == nil {
							basePorts = bp
							break
						}
					}
				}
			}
			projectBasePorts[pn] = basePorts

			for p, wt := range proj.Worktrees {
				assignments = append(assignments, assignment{pn, p, wt.Ports, basePorts})
			}
		}

		if len(assignments) == 0 {
			if len(config.AppConfig.Ports) == 0 && len(state.AppState.Projects) == 0 {
				cmd.Println("No port assignments found and no ports configured.")
				return nil
			}

			if !showAllPorts && currentProject != "" {
				if len(config.AppConfig.Ports) > 0 {
					cmd.Printf("No port assignments found for project '%s'.\n", currentProject)
				} else {
					cmd.Printf("No port assignments found for project '%s'. Use --all to see all projects.\n", currentProject)
					return nil
				}
			} else if showAllPorts && len(state.AppState.Projects) == 0 {
				cmd.Println("No port assignments found.")
			}
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

		// Header
		if len(assignments) > 0 {
			fmt.Fprint(w, "PROJECT\tWORKTREE PATH")
			for _, name := range portNames {
				fmt.Fprintf(w, "\t%s", strings.ToUpper(name))
			}
			fmt.Fprintln(w)

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
						basePort := 0
						if bp, ok := a.basePorts[name]; ok {
							basePort = bp
						}

						if basePort == 0 {
							fmt.Fprint(w, "\t-")
						} else {
							fmt.Fprintf(w, "\t%d", basePort+idx)
						}
					} else {
						fmt.Fprint(w, "\t-")
					}
				}
				fmt.Fprintln(w)
			}
			w.Flush()
		}

		// Base Ports Configuration
		if len(config.AppConfig.Ports) > 0 {
			cmd.Printf("\nBase Ports Configuration (current project: %s):\n", currentProject)
			var currentPortNames []string
			for name := range config.AppConfig.Ports {
				currentPortNames = append(currentPortNames, name)
			}
			sort.Strings(currentPortNames)
			for _, name := range currentPortNames {
				cmd.Printf("  - %s: %d\n", name, config.AppConfig.Ports[name])
			}
		}

		return nil
	},
}

var portsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update port assignments based on current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = state.LoadState()
		state.CleanupState()

		mainRoot, err := gitroot.GetMainProjectRoot()
		if err != nil {
			return logger.Errorf("not a git repository: %v", err)
		}
		projectName := filepath.Base(mainRoot)

		proj, ok := state.AppState.Projects[projectName]
		if !ok {
			cmd.Printf("No assignments found for project '%s'.\n", projectName)
			return nil
		}

		updatedCount := 0
		for path, wt := range proj.Worktrees {
			changed := false
			// Add missing ports from config
			for name, basePort := range config.AppConfig.Ports {
				if _, exists := wt.Ports[name]; !exists {
					state.AssignPortIndex(projectName, path, name, basePort)
					changed = true
				}
			}

			// Remove ports no longer in config
			for name := range wt.Ports {
				if _, inConfig := config.AppConfig.Ports[name]; !inConfig {
					delete(wt.Ports, name)
					changed = true
				}
			}

			if changed {
				updatedCount++
			}
		}

		if updatedCount > 0 {
			if err := state.SaveState(); err != nil {
				return logger.Errorf("failed to save state: %v", err)
			}
			cmd.Printf("Successfully updated %d worktree(s) in project '%s'.\n", updatedCount, projectName)
		} else {
			cmd.Printf("All worktrees in project '%s' are up to date.\n", projectName)
		}

		return nil
	},
}

func init() {
	portsCmd.Flags().BoolVarP(&showAllPorts, "all", "a", false, "show port assignments for all projects")
	portsCmd.AddCommand(portsUpdateCmd)
	rootCmd.AddCommand(portsCmd)
}
