package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/hook"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/template"
	"github.com/mocyuto/zgt/internal/tmux"
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
worktree path is automatically determined based on the main repository root.
If two arguments are provided, the first is the target path and the second is the branch.

Both forms will automatically create the branch if it does not already exist.`,
	Example: `  # Automated path: if repo root is 'path-to/myapp', creates worktree at 'path-to/myapp-feat'
  zgt add feat

  # Explicit path:
  zgt add ./experimental-worktree feat`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetPath, branch string
		if len(args) == 1 {
			branch = args[0]
			mainRoot, err := gitroot.GetMainProjectRoot()
			if err != nil {
				return logger.Errorf("failed to get main project root: %v", err)
			}
			projectName := filepath.Base(mainRoot)
			targetPath = filepath.Join(filepath.Dir(mainRoot), fmt.Sprintf("%s-%s", projectName, branch))
			logger.Info("Automated path: %s", targetPath)
		} else {
			targetPath = args[0]
			branch = args[1]
		}

		sourceRoot, err := gitroot.GetGitRoot()
		if err != nil {
			return logger.Errorf("failed to get git root: %v", err)
		}

		var baseBranch string
		if config.AppConfig.Add.FromDefault {
			defaultBranch, err := git.GetDefaultBranch()
			if err != nil {
				return logger.Errorf("failed to get default branch: %v", err)
			}
			baseBranch = defaultBranch
			logger.Info("Using default branch '%s' as base", baseBranch)

			if config.AppConfig.Add.AutoPull {
				logger.Info("Updating branch '%s'...", baseBranch)
				if err := git.Pull(baseBranch, baseBranch); err != nil {
					logger.Warn("pull failed: %v", err)
				} else {
					logger.Success("Successfully updated '%s'", baseBranch)
				}
			}
		}

		logger.Info("--- Creating worktree at %s ---", targetPath)
		if err := git.CreateWorktree(targetPath, branch, baseBranch); err != nil {
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
		gitRoot, _ := gitroot.GetMainProjectRoot()
		projectName := filepath.Base(gitRoot)
		for name, basePort := range config.AppConfig.Ports {
			state.AssignPortIndex(projectName, absPath, name, basePort)
		}
		_ = state.SaveState()

		// Run tmux setup
		mainRoot, _ := gitroot.GetMainProjectRoot()
		currentBranch, _ := git.GetCurrentBranch()
		if err := tmux.Setup(template.Context{
			Path:          absPath,
			Branch:        branch,
			Repo:          filepath.Base(mainRoot),
			CurrentDir:    filepath.Base(sourceRoot),
			TargetBranch:  branch,
			CurrentBranch: currentBranch,
		}); err != nil {
			logger.Warn("tmux setup failed: %v", err)
		}

		// Run add hooks
		hook.RunHooks("add", template.Context{
			Path:          targetPath,
			Branch:        branch,
			Repo:          filepath.Base(mainRoot),
			CurrentDir:    filepath.Base(sourceRoot),
			TargetBranch:  branch,
			CurrentBranch: currentBranch,
		})

		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&newBranch, "branch", "b", "", "create and checkout a new branch")
	addCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output")
	rootCmd.AddCommand(addCmd)
}
