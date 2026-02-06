package main

import (
	"os"
	"os/exec"
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
