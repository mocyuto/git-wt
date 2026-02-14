package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCmd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zgt-init-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to tmpDir
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Mocking Git Root for the test
	// In the real app, it calls git.GetMainProjectRoot()
	// Since we are in a fresh temp dir, it will fallback to "." as seen in init.go

	// Run init command logic (manually triggering the Run logic or similar)
	// For simplicity, we can test the helper functions or the core logic if exported.
	// Since we want to test the command, we'll try to run it.

	initCmd.Run(initCmd, []string{})

	// Check if zgt.config.yml was created
	configPath := filepath.Join(tmpDir, "zgt.config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("zgt.config.yml was not created")
	}

	// Check if .gitignore was created and contains config files
	ignorePath := filepath.Join(tmpDir, ".gitignore")
	data, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Errorf("failed to read .gitignore: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "zgt.config.yml") {
		t.Errorf(".gitignore does not contain zgt.config.yml")
	}
	if !strings.Contains(content, "zgt.config.yaml") {
		t.Errorf(".gitignore does not contain zgt.config.yaml")
	}

	// Test appending to existing .gitignore
	err = os.WriteFile(ignorePath, []byte("node_modules\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	initCmd.Run(initCmd, []string{})

	data, err = os.ReadFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	content = string(data)
	if !strings.Contains(content, "node_modules") {
		t.Errorf(".gitignore lost existing content")
	}

	// Count occurrences
	if strings.Count(content, "zgt.config.yml") > 1 {
		t.Errorf("zgt.config.yml was appended multiple times")
	}
}
