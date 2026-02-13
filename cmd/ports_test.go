package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/state"
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
		state.AppState.Projects = make(map[string]state.ProjectState)
		state.SaveState()

		buf := new(bytes.Buffer)
		portsCmd.SetOut(buf)

		err := portsCmd.RunE(portsCmd, []string{})
		if err != nil {
			t.Fatalf("RunE failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "No port assignments found") {
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
		state.AppState.Projects = map[string]state.ProjectState{
			"git-wt": {
				Worktrees: map[string]*state.WorktreeState{
					wt1: {Ports: map[string]int{"api": 0}},
					wt2: {Ports: map[string]int{"api": 1}},
				},
			},
		}
		state.SaveState()

		buf := new(bytes.Buffer)
		portsCmd.SetOut(buf)

		err = portsCmd.RunE(portsCmd, []string{})
		if err != nil {
			t.Fatalf("RunE failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "PROJECT") || !strings.Contains(output, "WORKTREE PATH") {
			t.Errorf("Expected header not found in output: %s", output)
		}
		if !strings.Contains(output, "git-wt") || !strings.Contains(output, wt1) {
			t.Errorf("Expected assignment for wt1 not found: %s", output)
		}
		if !strings.Contains(output, "git-wt") || !strings.Contains(output, wt2) {
			t.Errorf("Expected assignment for wt2 not found: %s", output)
		}
	})

	t.Run("With configured base ports", func(t *testing.T) {
		config.AppConfig.Ports = map[string]int{
			"api": 8080,
			"web": 3000,
		}

		// Create dummy worktree directories so CleanupState doesn't remove them
		wt1 := filepath.Join(tmpHome, "wt1")
		wt2 := filepath.Join(tmpHome, "wt2")
		os.MkdirAll(wt1, 0755)
		os.MkdirAll(wt2, 0755)

		// Setup initial state
		state.AppState.Projects = map[string]state.ProjectState{
			"git-wt": {
				Worktrees: map[string]*state.WorktreeState{
					wt1: {Ports: map[string]int{"api": 0, "web": 0}},
					wt2: {Ports: map[string]int{"api": 1, "web": 1}},
				},
			},
		}
		state.SaveState()

		buf := new(bytes.Buffer)
		portsCmd.SetOut(buf)

		err = portsCmd.RunE(portsCmd, []string{})
		if err != nil {
			t.Fatalf("RunE failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Base Ports Configuration") {
			t.Errorf("Expected base ports section not found: %s", output)
		}
		if !strings.Contains(output, "api: 8080") {
			t.Errorf("Expected api port config not found: %s", output)
		}

		// Check table content for calculated ports
		if !strings.Contains(output, "8080") || !strings.Contains(output, "3000") {
			t.Errorf("Expected ports 8080/3000 for index 0 not found: %s", output)
		}
		if !strings.Contains(output, "8081") || !strings.Contains(output, "3001") {
			t.Errorf("Expected ports 8081/3001 for index 1 not found: %s", output)
		}
	})
}
