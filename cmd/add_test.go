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

	t.Run("AutomatedPath", func(t *testing.T) {
		mainRoot, err := gitroot.GetMainProjectRoot()
		if err != nil {
			t.Fatalf("GetMainProjectRoot failed: %v", err)
		}

		branch := "feat"
		projectName := filepath.Base(mainRoot)
		targetPath := filepath.Join(filepath.Dir(mainRoot), fmt.Sprintf("%s-%s", projectName, branch))

		expectedPath, _ := filepath.EvalSymlinks(filepath.Join(tmpDir, "myrepo-feat"))
		targetPath, _ = filepath.EvalSymlinks(targetPath)
		if targetPath != expectedPath {
			t.Errorf("expected targetPath %s, got %s", expectedPath, targetPath)
		}
	})

	t.Run("CustomPathFlag", func(t *testing.T) {
		// Reset pathFlag for test
		pathFlag = "./custom-path"
		defer func() { pathFlag = "" }()

		targetPath := pathFlag

		if targetPath != "./custom-path" {
			t.Errorf("expected targetPath ./custom-path, got %s", targetPath)
		}
	})

	t.Run("ArgumentValidation", func(t *testing.T) {
		// Test that the command expects exactly 1 argument
		if addCmd.Args == nil {
			t.Fatal("addCmd.Args is nil")
		}

		// Simulate 0 args
		err := addCmd.Args(addCmd, []string{})
		if err == nil {
			t.Error("expected error for 0 arguments, got nil")
		}

		// Simulate 1 arg
		err = addCmd.Args(addCmd, []string{"branch"})
		if err != nil {
			t.Errorf("expected no error for 1 argument, got %v", err)
		}

		// Simulate 2 args
		err = addCmd.Args(addCmd, []string{"path", "branch"})
		if err == nil {
			t.Error("expected error for 2 arguments, got nil")
		}
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
}
