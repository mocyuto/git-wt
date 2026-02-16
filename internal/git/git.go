package git

import (
	"os"
	"os/exec"
	"strings"

	"github.com/mocyuto/zgt/internal/logger"
)

func GetCurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func BranchExists(branch string) bool {
	// Check local branches
	err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	if err == nil {
		return true
	}
	// Check remote branches
	err = exec.Command("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch).Run()
	return err == nil
}

func GetDefaultBranch() (string, error) {
	out, err := exec.Command("git", "remote", "show", "origin").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "HEAD branch:") {
				return strings.TrimSpace(strings.Split(line, ":")[1]), nil
			}
		}
	}

	// Fallback to git config or common names
	out, err = exec.Command("git", "config", "init.defaultBranch").Output()
	if err == nil && len(out) > 0 {
		return strings.TrimSpace(string(out)), nil
	}

	if BranchExists("main") {
		return "main", nil
	}
	return "master", nil
}

func Pull(remoteBranch, localBranch string) error {
	cmd := exec.Command("git", "pull", "origin", remoteBranch+":"+localBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CreateWorktree(path, branch, base string) error {
	cmdArgs := []string{"worktree", "add", path}
	if !BranchExists(branch) {
		cmdArgs = append(cmdArgs, "-b", branch)
		if base != "" {
			cmdArgs = append(cmdArgs, base)
		}
	} else {
		cmdArgs = append(cmdArgs, branch)
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RemoveWorktree(path string, force bool) error {
	cmdArgs := []string{"worktree", "remove", path}
	if force {
		cmdArgs = append(cmdArgs, "--force")
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ResolveWorktreeInfo(search string) (path, branch string, err error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(string(out), "\n")
	var currentPath string
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			fullBranch := strings.TrimPrefix(line, "branch ")
			branchName := strings.TrimPrefix(fullBranch, "refs/heads/")

			// Match by branch name or path
			if branchName == search || currentPath == search || strings.HasSuffix(fullBranch, "/"+search) {
				return currentPath, branchName, nil
			}
		}
	}
	return "", "", logger.Errorf("not found")
}
