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

func PullWithAutostash(dir, branch string) error {
	stashed := false
	uncommitted, err := HasUncommittedFilesDir(dir)
	if err != nil {
		logger.Warn("failed to check for uncommitted files in %s: %v", dir, err)
	} else if uncommitted {
		logger.Info("Stashing uncommitted changes in %s before pull...", dir)
		if err := StashInDir(dir, "zgt: autostash before autopull"); err != nil {
			logger.Warn("stash failed in %s: %v", dir, err)
		} else {
			stashed = true
		}
	}

	logger.Info("Updating branch '%s' in %s...", branch, dir)
	var pullErr error
	if dir == "" || dir == "." {
		pullErr = Pull(branch, branch)
	} else {
		pullErr = PullInDir(dir, branch)
	}

	if pullErr != nil {
		if stashed {
			logger.Info("Restoring uncommitted changes in %s after failed pull...", dir)
			if err := StashPopInDir(dir); err != nil {
				logger.Warn("stash pop failed in %s: %v", dir, err)
			}
		}
		return pullErr
	}

	if stashed {
		logger.Info("Restoring uncommitted changes in %s...", dir)
		if err := StashPopInDir(dir); err != nil {
			logger.Warn("stash pop failed in %s: %v", dir, err)
		}
	}
	return nil
}

func PullInDir(dir, branch string) error {
	cmd := exec.Command("git", "pull", "origin", branch)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Fetch(branch string) error {
	cmd := exec.Command("git", "fetch", "origin", branch)
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

func GetBranchWorktree(branch string) (string, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	var currentPath string
	for _, line := range lines {
		if after, ok := strings.CutPrefix(line, "worktree "); ok {
			currentPath = after
		} else if after, ok := strings.CutPrefix(line, "branch "); ok {
			fullBranch := after
			branchName := strings.TrimPrefix(fullBranch, "refs/heads/")
			if branchName == branch {
				return currentPath, nil
			}
		}
	}
	return "", logger.Errorf("branch not checked out")
}

func GetWorktreeCompletions() ([]string, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var completions []string
	for _, line := range lines {
		if after, ok := strings.CutPrefix(line, "branch "); ok {
			fullBranch := after
			branchName := strings.TrimPrefix(fullBranch, "refs/heads/")
			completions = append(completions, branchName)
		}
	}
	return completions, nil
}

func HasUncommittedFiles() (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func HasUncommittedFilesDir(dir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func Stash(message string) error {
	cmd := exec.Command("git", "stash", "push", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func StashInDir(dir, message string) error {
	cmd := exec.Command("git", "stash", "push", "-m", message)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func StashPop() error {
	cmd := exec.Command("git", "stash", "pop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func StashPopInDir(dir string) error {
	cmd := exec.Command("git", "stash", "pop")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
