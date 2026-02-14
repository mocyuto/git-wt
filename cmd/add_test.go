package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mocyuto/zgt/internal/gitroot"
)

func TestAddPathLogic(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "zgt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup a mock project structure:
	// tmpDir/
	//   myrepo/ (root)
	//     .git/
	//     cmd/
	repoRoot := filepath.Join(tmpDir, "myrepo")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}
	cmdDir := filepath.Join(repoRoot, "cmd")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatalf("failed to create cmd dir: %v", err)
	}

	// Change to cmdDir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	if err := os.Chdir(cmdDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Init a real git repo in the temp dir to make gitroot work
	runGit(t, repoRoot, "init")

	mainRoot, err := gitroot.GetMainProjectRoot()
	if err != nil {
		t.Fatalf("GetMainProjectRoot failed: %v", err)
	}

	if !strings.HasSuffix(mainRoot, "myrepo") {
		t.Errorf("expected mainRoot to end with 'myrepo', got %s", mainRoot)
	}

	branch := "feat"
	projectName := filepath.Base(mainRoot)
	targetPath := filepath.Join(filepath.Dir(mainRoot), fmt.Sprintf("%s-%s", projectName, branch))

	expectedPath, _ := filepath.EvalSymlinks(filepath.Join(tmpDir, "myrepo-feat"))
	targetPath, _ = filepath.EvalSymlinks(targetPath)
	if targetPath != expectedPath {
		t.Errorf("expected targetPath %s, got %s", expectedPath, targetPath)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
}
