package main

import (
	"fmt"
	"os"
	"path/filepath"

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
