package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees with PR status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return ListWorktrees()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func ListWorktrees() error {
	wts, err := git.GetWorktrees()
	if err != nil {
		return err
	}

	prMap := make(map[string]git.PRInfo)
	if git.HasGh() {
		prs, err := git.GetPRs()
		if err == nil {
			for _, pr := range prs {
				// Keep the latest PR for the same branch
				if existing, ok := prMap[pr.HeadRefName]; !ok || pr.Number > existing.Number {
					prMap[pr.HeadRefName] = pr
				}
			}
		}
	}

	_ = state.LoadState()
	portsMap := make(map[string]map[string]int)
	// We don't have a direct way to get all ports by path easily from State
	// Let's just use GetCurrentWorktreePorts logic if we were in that dir,
	// but since we iterate all worktrees, let's just pre-process AppState.
	for _, proj := range state.AppState.Projects {
		for p, wt := range proj.Worktrees {
			portsMap[p] = wt.Ports
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tCOMMIT\tBRANCH\tSTATUS\tPORTS")

	for _, wt := range wts {
		// Check for local changes
		hasChanges, _ := git.CheckLocalChanges(wt.Path)
		wt.HasLocalChanges = hasChanges

		head := wt.HEAD
		if len(head) > 7 {
			head = head[:7]
		}

		branchName := ""
		if wt.Branch != "" {
			branchName = strings.TrimPrefix(wt.Branch, "refs/heads/")
			if pr, ok := prMap[branchName]; ok {
				status := strings.ToUpper(pr.State)
				if pr.IsDraft {
					status = "DRAFT"
				}
				branchName += fmt.Sprintf(" (PR:#%d [%s])", pr.Number, status)
			}
		}

		portStr := "-"
		if ports, ok := portsMap[wt.Path]; ok {
			var portInfos []string
			for name, idx := range ports {
				basePort := config.AppConfig.Ports[name]
				portInfos = append(portInfos, fmt.Sprintf("%s:%d", name, basePort+idx))
			}
			if len(portInfos) > 0 {
				sort.Strings(portInfos)
				portStr = strings.Join(portInfos, ",")
			}
		}

		statusStr := ""
		if wt.HasLocalChanges {
			statusStr = "[DIRTY]"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", wt.Path, head, branchName, statusStr, portStr)
	}
	w.Flush()

	return nil
}
