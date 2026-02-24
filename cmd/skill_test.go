package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunInstall_Embedded(t *testing.T) {
	// Create a temporary directory and change to it
	tmpDir, err := os.MkdirTemp("", "zgt-skill-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(origCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	// Create local .claude/skills in tmpDir to simulate target
	targetDir := filepath.Join(tmpDir, ".claude", "skills")

	// Mock skillInstallAll to true to avoid TUI
	skillInstallAll = true
	defer func() { skillInstallAll = false }()

	// Run install
	if err := runInstall(); err != nil {
		t.Fatalf("Failed to run install: %v", err)
	}

	// Check if zgt-usage was installed
	skillPath := filepath.Join(targetDir, "zgt-usage")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("Expected skill 'zgt-usage' to be installed at %q, but it was not found", skillPath)
	}
}
