package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/git"
	"github.com/mocyuto/git-wt/internal/hook"
	"github.com/mocyuto/git-wt/internal/logger"
	"github.com/mocyuto/git-wt/internal/state"
	"github.com/mocyuto/git-wt/internal/template"
	"github.com/spf13/cobra"
)

var (
	newBranch string
	verbose   bool
)

var addCmd = &cobra.Command{
	Use:   "add [path] <branch>",
	Short: "Create git worktree and copy ignored files",
	Long: `Create a new git worktree, optionally creating a new branch, and
automatically copy ignored configuration files (like .env) from the main tree.

If only one argument is provided, it is treated as the branch name, and the
worktree path is automatically determined based on the current directory name.
If two arguments are provided, the first is the target path and the second is the branch.

Both forms will automatically create the branch if it does not already exist.`,
	Example: `  # Automated path: if current dir is 'myapp', creates worktree at '../myapp-feat'
  git-wt add feat

  # Explicit path:
  git-wt add ./experimental-worktree feat`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetPath, branch string
		if len(args) == 1 {
			// Automate path: ../{current_dir}-{branch}
			branch = args[0]

			// Auto-create branch if it doesn't exist
			if !git.BranchExists(branch) && newBranch == "" {
				newBranch = branch
				logger.Info("Branch '%s' does not exist. It will be created.", branch)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return logger.Errorf("failed to get current directory: %v", err)
			}
			projectName := filepath.Base(cwd)
			targetPath = filepath.Join("..", fmt.Sprintf("%s-%s", projectName, branch))
			logger.Info("Automated path: %s", targetPath)
		} else {
			targetPath = args[0]
			branch = args[1]

			// Auto-create branch if it doesn't exist
			if !git.BranchExists(branch) && newBranch == "" {
				newBranch = branch
				logger.Info("Branch '%s' does not exist. It will be created.", branch)
			}
		}

		sourceRoot, err := git.GetGitRoot()
		if err != nil {
			return logger.Errorf("failed to get git root: %v", err)
		}

		logger.Info("--- Creating worktree at %s ---", targetPath)
		if err := git.CreateWorktree(targetPath, newBranch, branch); err != nil {
			return logger.Errorf("error creating worktree: %v", err)
		}

		logger.Info("--- Copying ignored configuration files ---")
		if err := git.CopyIgnoredFiles(sourceRoot, targetPath, config.AppConfig.Ignore, verbose); err != nil {
			return logger.Errorf("error copying files: %v", err)
		}

		logger.Success("--- Done! ---")
		logger.Success("New worktree is ready at: %s", targetPath)

		// Assign port index
		absPath := state.NormalizePath(targetPath)
		_ = state.LoadState() // Assign ports
		gitRoot, _ := git.GetMainProjectRoot()
		projectName := filepath.Base(gitRoot)
		for name, basePort := range config.AppConfig.Ports {
			state.AssignPortIndex(projectName, absPath, name, basePort)
		}
		_ = state.SaveState()

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
