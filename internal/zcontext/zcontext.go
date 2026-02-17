package zcontext

import (
	"path/filepath"

	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/template"
)

// New creates a new template.Context with populated fields.
// path is the target path (e.g. worktree path).
// targetBranch is the branch being operated on.
func New(path, targetBranch string) template.Context {
	mainRoot, _ := gitroot.GetMainProjectRoot()
	currentRoot, _ := gitroot.GetGitRoot()
	currentBranch, _ := git.GetCurrentBranch()

	// If targetBranch is empty (e.g. in config or env commands), use currentBranch
	if targetBranch == "" {
		targetBranch = currentBranch
	}

	return template.Context{
		Path:          path,
		Branch:        targetBranch,
		Repo:          filepath.Base(mainRoot),
		CurrentDir:    filepath.Base(currentRoot),
		TargetBranch:  targetBranch,
		CurrentBranch: currentBranch,
	}
}
