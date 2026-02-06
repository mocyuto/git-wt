package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	newBranch string
)

var rootCmd = &cobra.Command{
	Use:   "git-wt",
	Short: "Create git worktree and copy ignored files",
	Long: `git-wt is a CLI tool that extends 'git worktree add' by automatically
copying ignored configuration files (like .env) from the main tree to the new worktree.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/git-wt/config.yaml)")
}

func GetGitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
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

func CreateWorktree(path, newBranch, branch string) error {
	cmdArgs := []string{"worktree", "add", path}
	if newBranch != "" {
		cmdArgs = append(cmdArgs, "-b", newBranch)
	} else if branch != "" {
		cmdArgs = append(cmdArgs, branch)
	}

	cmd := exec.Command("git", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func CopyIgnoredFiles(sourceRoot, targetPath string) error {
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	cmd.Dir = sourceRoot
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		relPath := scanner.Text()

		// Filter by ignore patterns in config
		ignored := false
		for _, pattern := range AppConfig.Ignore {
			match, err := filepath.Match(pattern, relPath)
			if err == nil && match {
				ignored = true
				break
			}
			// Also check if pattern matches a directory in relPath
			// filepath.Match doesn't handle directory matching like .gitignore automatically
			// For simplicity, we also check if any part of the path matches
			parts := strings.SplitSeq(relPath, string(os.PathSeparator))
			for part := range parts {
				match, err := filepath.Match(pattern, part)
				if err == nil && match {
					ignored = true
					break
				}
			}
			if ignored {
				break
			}
		}

		if ignored {
			fmt.Printf("Skipping ignored file: %s\n", relPath)
			continue
		}

		src := filepath.Join(sourceRoot, relPath)
		dst := filepath.Join(targetPath, relPath)

		info, err := os.Stat(src)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %v", dst, err)
		}

		if err := copyFile(src, dst); err != nil {
			fmt.Printf("Failed to copy %s: %v\n", relPath, err)
		} else {
			fmt.Printf("Copied: %s\n", relPath)
		}
	}

	return cmd.Wait()
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
