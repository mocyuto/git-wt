package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mocyuto/git-wt/internal/git"
	"github.com/mocyuto/git-wt/internal/state"
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
				prMap[pr.HeadRefName] = pr
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

	for _, wt := range wts {
		// Check for local changes
		hasChanges, _ := git.CheckLocalChanges(wt.Path)
		wt.HasLocalChanges = hasChanges

		head := wt.HEAD
		if len(head) > 7 {
			head = head[:7]
		}
		line := fmt.Sprintf("%-40s %s", wt.Path, head)
		if wt.Branch != "" {
			branchName := strings.TrimPrefix(wt.Branch, "refs/heads/")
			line += fmt.Sprintf(" [%s]", branchName)

			if pr, ok := prMap[branchName]; ok {
				status := strings.ToUpper(pr.State)
				if pr.IsDraft {
					status = "DRAFT"
				}
				line += fmt.Sprintf(" PR:#%d [%s]", pr.Number, status)
			}
		}

		if ports, ok := portsMap[wt.Path]; ok {
			var portInfos []string
			for name, idx := range ports {
				// We don't have basePort here easily without config,
				// but showing the index or the actual port if we had basePort would be better.
				// For now, let's just show key:index.
				portInfos = append(portInfos, fmt.Sprintf("%s:%d", name, idx))
			}
			if len(portInfos) > 0 {
				sort.Strings(portInfos)
				line += fmt.Sprintf(" ports:[%s]", strings.Join(portInfos, ","))
			}
		}

		if wt.HasLocalChanges {
			line += " [DIRTY]"
		}
		fmt.Println(line)
	}

	return nil
}
