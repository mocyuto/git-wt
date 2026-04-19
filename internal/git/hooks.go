package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ResolveHooksPath(projectRoot, configuredPath string) (string, error) {
	if configuredPath == "" {
		return "", fmt.Errorf("git hooks path is empty")
	}

	if filepath.IsAbs(configuredPath) {
		return filepath.Clean(configuredPath), nil
	}
	if projectRoot == "" {
		return "", fmt.Errorf("project root is empty")
	}

	return filepath.Join(projectRoot, configuredPath), nil
}

func RegisterHooksPath(worktreePath, hooksPath string) (bool, string, error) {
	resolvedWorktree, err := filepath.Abs(worktreePath)
	if err != nil {
		return false, "", err
	}
	resolvedHooks, err := filepath.Abs(hooksPath)
	if err != nil {
		return false, "", err
	}
	if info, err := os.Stat(resolvedHooks); err != nil {
		if os.IsNotExist(err) {
			return false, "", fmt.Errorf("git hooks path does not exist: %s", resolvedHooks)
		}
		return false, "", err
	} else if !info.IsDir() {
		return false, "", fmt.Errorf("git hooks path is not a directory: %s", resolvedHooks)
	}

	// Check if explicitly set in worktree config.
	// Inherited values from the main repository or global config should be overridden.
	current, err := getWorktreeConfigValue(resolvedWorktree, "core.hooksPath")
	if err != nil {
		return false, "", err
	}
	if current != "" {
		return false, current, nil
	}

	if err := runConfigSet(resolvedWorktree, []string{"config", "extensions.worktreeConfig", "true"}); err != nil {
		return false, "", err
	}
	if err := runConfigSet(resolvedWorktree, []string{"config", "--worktree", "core.hooksPath", resolvedHooks}); err != nil {
		return false, "", err
	}

	return true, "", nil
}

func getWorktreeConfigValue(dir, key string) (string, error) {
	// First, check if worktree config is enabled
	enabled, err := getConfigValue(dir, "extensions.worktreeConfig")
	if err != nil || enabled != "true" {
		return "", nil // Not enabled means no worktree-specific config exists yet
	}

	cmd := exec.Command("git", "config", "--get", "--worktree", key)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if strings.TrimSpace(string(out)) == "" && errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func getConfigValue(dir, key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if strings.TrimSpace(string(out)) == "" && errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func runConfigSet(dir string, args []string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}
