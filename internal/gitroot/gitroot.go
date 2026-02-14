package gitroot

import (
	"os"
	"os/exec"
	"strings"
)

// GetGitRoot returns the absolute path to the root of the current git repository.
func GetGitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetMainProjectRoot returns the absolute path to the main project root directory.
// It is a wrapper around GetMainProjectRootFromPath using the current working directory.
func GetMainProjectRoot() (string, error) {
	return GetMainProjectRootFromPath("")
}

// GetMainProjectRootFromPath returns the absolute path to the main project root for the given directory.
// If the directory is part of a git worktree, it returns the main repository's working directory
// rather than the worktree path.
func GetMainProjectRootFromPath(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(out))
	// If it's a worktree, git-common-dir returns the .git directory of the main project.
	// We want the working directory of the main project.
	if strings.HasSuffix(path, "/.git") {
		return strings.TrimSuffix(path, "/.git"), nil
	}
	if strings.HasSuffix(path, string(os.PathSeparator)+".git") {
		return strings.TrimSuffix(path, string(os.PathSeparator)+".git"), nil
	}
	return path, nil
}
