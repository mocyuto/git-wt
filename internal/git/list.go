package git

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

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

func GetWorktrees() ([]Worktree, error) {
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

func HasGh() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func GetPRs() ([]PRInfo, error) {
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

// CheckLocalChanges checks if the worktree has uncommitted changes
func CheckLocalChanges(wtPath string) (bool, error) {
	cmd := exec.Command("git", "-C", wtPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}
