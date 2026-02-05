package cmd

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
	showList  bool
)

var rootCmd = &cobra.Command{
	Use:   "git-wt <path> [<branch>]",
	Short: "Create git worktree and copy ignored files",
	Long: `git-wt is a CLI tool that extends 'git worktree add' by automatically
copying ignored configuration files (like .env) from the main tree to the new worktree.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showList {
			return ListWorktrees()
		}

		if len(args) < 1 {
			return cmd.Help()
		}

		targetPath := args[0]
		var branch string
		if len(args) > 1 {
			branch = args[1]
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
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&newBranch, "branch", "b", "", "create and checkout a new branch")
	rootCmd.Flags().BoolVarP(&showList, "list", "l", false, "list all worktrees")
}

func GetGitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
