package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) (string, string) {
	tempDir, err := os.MkdirTemp("", "git-wt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	repoDir := filepath.Join(tempDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test User")

	// Create .gitignore and some ignored files
	gitignoreContent := ".env\nnode_modules/\n"
	if err := os.WriteFile(filepath.Join(repoDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to write .gitignore: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoDir, ".env"), []byte("SECRET=123"), 0644); err != nil {
		t.Fatalf("Failed to write .env: %v", err)
	}

	nmDir := filepath.Join(repoDir, "node_modules")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "config.json"), []byte(`{"debug":true}`), 0644); err != nil {
		t.Fatalf("Failed to write node_modules/config.json: %v", err)
	}

	runGit(t, repoDir, "add", ".gitignore")
	runGit(t, repoDir, "commit", "-m", "initial commit")

	return tempDir, repoDir
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(out))
	}
}

func TestCopyIgnoredFiles(t *testing.T) {
	tempDir, repoDir := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	targetDir := filepath.Join(tempDir, "worktree")

	// Pre-create worktree dir (usually done by git worktree add, but we test copyIgnoredFiles individually)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	err := copyIgnoredFiles(repoDir, targetDir)
	if err != nil {
		t.Errorf("copyIgnoredFiles failed: %v", err)
	}

	// Verify .env
	envPath := filepath.Join(targetDir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf(".env was not copied")
	}

	// Verify node_modules/config.json
	configPath := filepath.Join(targetDir, "node_modules", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("node_modules/config.json was not copied")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Errorf("Failed to read copied file: %v", err)
	}
	if string(content) != `{"debug":true}` {
		t.Errorf("Content mismatch: got %s, want %s", string(content), `{"debug":true}`)
	}
}
