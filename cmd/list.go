package cmd

import (
	"fmt"
	"strings"

	"github.com/mocyuto/git-wt/internal/git"
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
		if wt.HasLocalChanges {
			line += " [DIRTY]"
		}
		fmt.Println(line)
	}

	return nil
}
