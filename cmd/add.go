package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/git"
	"github.com/mocyuto/git-wt/internal/hook"
	"github.com/mocyuto/git-wt/internal/state"
	"github.com/mocyuto/git-wt/internal/template"
	"github.com/spf13/cobra"
)

var (
	newBranch string
	verbose   bool
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
			if !git.BranchExists(branch) && newBranch == "" {
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
			if !git.BranchExists(branch) && newBranch == "" {
				newBranch = branch
				fmt.Printf("Branch '%s' does not exist. It will be created.\n", branch)
			}
		}

		sourceRoot, err := git.GetGitRoot()
		if err != nil {
			return fmt.Errorf("failed to get git root: %v", err)
		}

		fmt.Printf("--- Creating worktree at %s ---\n", targetPath)
		if err := git.CreateWorktree(targetPath, newBranch, branch); err != nil {
			return fmt.Errorf("error creating worktree: %v", err)
		}

		fmt.Println("--- Copying ignored configuration files ---")
		if err := git.CopyIgnoredFiles(sourceRoot, targetPath, config.AppConfig.Ignore, verbose); err != nil {
			return fmt.Errorf("error copying files: %v", err)
		}

		fmt.Println("--- Done! ---")
		fmt.Printf("New worktree is ready at: %s\n", targetPath)

		// Assign port index
		absPath, _ := filepath.Abs(targetPath)
		_ = state.LoadState()
		idx := state.AssignPortIndex(absPath)
		_ = state.SaveState()
		fmt.Printf("Assigned Port Index: %d\n", idx)

		// Run add hooks
		hook.RunHooks("add", template.Context{
			Path:   targetPath,
			Branch: branch,
			Repo:   filepath.Base(sourceRoot),
		})

		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&newBranch, "branch", "b", "", "create and checkout a new branch")
	addCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output")
	rootCmd.AddCommand(addCmd)
}
