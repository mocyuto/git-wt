package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortsCmd(t *testing.T) {
	// Setup temporary home
	tmpHome, err := os.MkdirTemp("", "git-wt-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	t.Run("Empty state", func(t *testing.T) {
		AppState.Worktrees = make(map[string]int)
		SaveState()

		buf := new(bytes.Buffer)
		portsCmd.SetOut(buf)

		err := portsCmd.RunE(portsCmd, []string{})
		if err != nil {
			t.Fatalf("RunE failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "No port assignments found.") {
			t.Errorf("Expected empty state message not found: %s", output)
		}
	})

	t.Run("With assignments", func(t *testing.T) {
		// Create dummy worktree directories so CleanupState doesn't remove them
		wt1 := filepath.Join(tmpHome, "wt1")
		wt2 := filepath.Join(tmpHome, "wt2")
		os.MkdirAll(wt1, 0755)
		os.MkdirAll(wt2, 0755)

		// Setup initial state
		AppState.Worktrees = map[string]int{
			wt1: 0,
			wt2: 1,
		}
		SaveState()

		buf := new(bytes.Buffer)
		portsCmd.SetOut(buf)

		err = portsCmd.RunE(portsCmd, []string{})
		if err != nil {
			t.Fatalf("RunE failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "INDEX  WORKTREE PATH") {
			t.Errorf("Expected header not found in output: %s", output)
		}
		if !strings.Contains(output, "0      "+wt1) {
			t.Errorf("Expected assignment for wt1 not found: %s", output)
		}
		if !strings.Contains(output, "1      "+wt2) {
			t.Errorf("Expected assignment for wt2 not found: %s", output)
		}
	})

	t.Run("With configured base ports", func(t *testing.T) {
		AppConfig.Ports = map[string]int{
			"api": 8080,
		}

		buf := new(bytes.Buffer)
		portsCmd.SetOut(buf)

		err = portsCmd.RunE(portsCmd, []string{})
		if err != nil {
			t.Fatalf("RunE failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Configured Base Ports:") {
			t.Errorf("Expected base ports section not found: %s", output)
		}
		if !strings.Contains(output, "api: 8080") {
			t.Errorf("Expected api port config not found: %s", output)
		}
	})
}
