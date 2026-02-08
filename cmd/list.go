package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

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

type PRInfo struct {
	Number      int    `json:"number"`
	State       string `json:"state"`
	IsDraft     bool   `json:"isDraft"`
	HeadRefName string `json:"headRefName"`
}

type Worktree struct {
	Path            string
	HEAD            string
	Branch          string
	HasLocalChanges bool
}

func ListWorktrees() error {
	wts, err := getWorktrees()
	if err != nil {
		return err
	}

	prMap := make(map[string]PRInfo)
	if hasGh() {
		prs, err := getPRs()
		if err == nil {
			for _, pr := range prs {
				prMap[pr.HeadRefName] = pr
			}
		}
	}

	for _, wt := range wts {
		// Check for local changes
		hasChanges, _ := checkLocalChanges(wt.Path)
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

func getWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %v", err)
	}

	var wts []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current != nil {
				wts = append(wts, *current)
				current = nil
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "worktree":
			if current != nil {
				wts = append(wts, *current)
			}
			current = &Worktree{Path: value}
		case "HEAD":
			if current != nil {
				current.HEAD = value
			}
		case "branch":
			if current != nil {
				current.Branch = value
			}
		}
	}
	if current != nil {
		wts = append(wts, *current)
	}

	return wts, nil
}

func hasGh() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func getPRs() ([]PRInfo, error) {
	cmd := exec.Command("gh", "pr", "list", "--state", "all", "--limit", "100", "--json", "number,state,isDraft,headRefName")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var prs []PRInfo
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

// checkLocalChanges checks if the worktree has uncommitted changes
func checkLocalChanges(wtPath string) (bool, error) {
	cmd := exec.Command("git", "-C", wtPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}
