package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	var newBranch string
	flag.StringVar(&newBranch, "b", "", "create a new branch")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: git-wt [-b <new-branch>] <path> [<branch>]\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  <path>    New worktree path\n")
		fmt.Fprintf(os.Stderr, "  -b        Create and checkout a new branch (must come before <path>)\n")
		fmt.Fprintf(os.Stderr, "  <branch>  Existing branch to checkout\n")
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	targetPath := args[0]
	var branch string
	if len(args) > 1 {
		branch = args[1]
	}

	sourceRoot, err := getGitRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("--- Creating worktree at %s ---\n", targetPath)
	if err := createWorktree(targetPath, newBranch, branch); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating worktree: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- Copying ignored configuration files ---")
	if err := copyIgnoredFiles(sourceRoot, targetPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- Done! ---")
	fmt.Printf("New worktree is ready at: %s\n", targetPath)
}

func getGitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func createWorktree(path, newBranch, branch string) error {
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

func copyIgnoredFiles(sourceRoot, targetPath string) error {
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
		src := filepath.Join(sourceRoot, relPath)
		dst := filepath.Join(targetPath, relPath)

		info, err := os.Stat(src)
		if err != nil {
			continue // Skip if file doesn't exist (e.g. directory list)
		}
		if info.IsDir() {
			continue // git ls-files usually returns files, but be safe
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
