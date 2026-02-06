package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <path> [<branch>]",
	Short: "Create git worktree and copy ignored files",
	Long: `Create a new git worktree, optionally creating a new branch, and
automatically copy ignored configuration files (like .env) from the main tree.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetPath, branch string
		if len(args) == 1 {
			// Automate path: ../{current_dir}-{branch}
			branch = args[0]

			// Auto-create branch if it doesn't exist
			if !BranchExists(branch) && newBranch == "" {
				newBranch = branch
				fmt.Printf("Branch '%s' does not exist. It will be created.\n", branch)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}
			projectName := filepath.Base(cwd)
			targetPath = filepath.Join("..", fmt.Sprintf("%s-%s", projectName, branch))
			fmt.Printf("Automated path: %s\n", targetPath)
		} else {
			targetPath = args[0]
			branch = args[1]

			// Auto-create branch if it doesn't exist
			if !BranchExists(branch) && newBranch == "" {
				newBranch = branch
				fmt.Printf("Branch '%s' does not exist. It will be created.\n", branch)
			}
		}

		sourceRoot, err := GetGitRoot()
		if err != nil {
			return fmt.Errorf("failed to get git root: %v", err)
		}

		fmt.Printf("--- Creating worktree at %s ---\n", targetPath)
		if err := CreateWorktree(targetPath, newBranch, branch); err != nil {
			return fmt.Errorf("error creating worktree: %v", err)
		}

		fmt.Println("--- Copying ignored configuration files ---")
		if err := CopyIgnoredFiles(sourceRoot, targetPath); err != nil {
			return fmt.Errorf("error copying files: %v", err)
		}

		fmt.Println("--- Done! ---")
		fmt.Printf("New worktree is ready at: %s\n", targetPath)

		// Run add hooks
		RunHooks("add", HookContext{
			Path:   targetPath,
			Branch: branch,
			Repo:   filepath.Base(sourceRoot),
		})

		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&newBranch, "branch", "b", "", "create and checkout a new branch")
	rootCmd.AddCommand(addCmd)
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

	// Collect all files to copy first
	var filesToCopy []string
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

		// Check if it's a file (not directory)
		src := filepath.Join(sourceRoot, relPath)
		info, err := os.Stat(src)
		if err != nil || info.IsDir() {
			continue
		}

		filesToCopy = append(filesToCopy, relPath)
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	// Use worker pool pattern to limit concurrency
	const maxWorkers = 20
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, relPath := range filesToCopy {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(relPath string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			src := filepath.Join(sourceRoot, relPath)
			dst := filepath.Join(targetPath, relPath)

			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to create directory for %s: %v", dst, err)
				}
				mu.Unlock()
				return
			}

			if err := copyFile(src, dst); err != nil {
				fmt.Printf("Failed to copy %s: %v\n", relPath, err)
			} else {
				fmt.Printf("Copied: %s\n", relPath)
			}
		}(relPath)
	}

	wg.Wait()

	return firstErr
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
