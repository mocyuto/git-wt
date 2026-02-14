package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/state"
)

func TestPortsCmd(t *testing.T) {
	// Setup temporary home
	tmpHome, err := os.MkdirTemp("", "zgt-test-home")
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
		mainRoot, _ := git.GetMainProjectRoot()
		projectName := filepath.Base(mainRoot)

		state.AppState.Projects = map[string]state.ProjectState{
			projectName: {
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
		if !strings.Contains(output, projectName) || !strings.Contains(output, wt1) {
			t.Errorf("Expected assignment for wt1 not found: %s", output)
		}
		if !strings.Contains(output, projectName) || !strings.Contains(output, wt2) {
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
		mainRoot, _ := git.GetMainProjectRoot()
		projectName := filepath.Base(mainRoot)

		state.AppState.Projects = map[string]state.ProjectState{
			projectName: {
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

	t.Run("Filtering of port keys", func(t *testing.T) {
		mainRoot, _ := git.GetMainProjectRoot()
		currentProject := filepath.Base(mainRoot)
		t.Logf("Detected current project: %s", currentProject)

		config.AppConfig.Ports = map[string]int{
			"api": 8080,
		}

		pa1 := filepath.Join(tmpHome, "pa1")
		pb1 := filepath.Join(tmpHome, "pb1")
		os.MkdirAll(pa1, 0755)
		os.MkdirAll(pb1, 0755)

		// Current project: "project-a" (should match currentProject)
		// Another project in state: "project-b" with "front" port
		state.AppState.Projects = map[string]state.ProjectState{
			currentProject: {
				Worktrees: map[string]*state.WorktreeState{
					pa1: {Ports: map[string]int{"api": 0}},
				},
			},
			"other-project": {
				Worktrees: map[string]*state.WorktreeState{
					pb1: {Ports: map[string]int{"front": 0}},
				},
			},
		}
		state.SaveState()

		// Mock GetMainProjectRoot to return project-a's root
		// We actually rely on filepath.Base(mainRoot) in the command.
		// Since we are running in a test, let's see how we can control 'currentProject'.
		// cmd/ports.go:
		// mainRoot, err := git.GetMainProjectRoot()
		// currentProject = filepath.Base(mainRoot)

		// This is hard to mock without refactoring git.go.
		// But in this test environment, GetMainProjectRoot might fail or return something else.
		// If it fails, currentProject is "".

		buf := new(bytes.Buffer)
		portsCmd.SetOut(buf)

		// If currentProject is "", all ports should be collected if we don't handle it.
		// But the refined logic says:
		// else if currentProject != "" { ... }

		// Let's assume currentProject is "" for now (as it might be in test depending on where it runs).
		// If currentProject is "", then only config ports are shown.

		err = portsCmd.RunE(portsCmd, []string{})
		output := buf.String()

		// Should contain API but NOT FRONT
		if !strings.Contains(output, "API") {
			t.Errorf("Expected API column, got: %s", output)
		}
		if strings.Contains(output, "FRONT") {
			t.Errorf("Did not expect FRONT column, got: %s", output)
		}

		// Now with --all
		showAllPorts = true // set flag
		defer func() { showAllPorts = false }()

		buf = new(bytes.Buffer)
		portsCmd.SetOut(buf)
		err = portsCmd.RunE(portsCmd, []string{})
		output = buf.String()

		// Should contain both
		if !strings.Contains(output, "API") || !strings.Contains(output, "FRONT") {
			t.Errorf("With --all, expected both API and FRONT columns, got: %s", output)
		}
	})
}
