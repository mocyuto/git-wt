package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mocyuto/git-wt/internal/config"
	"github.com/mocyuto/git-wt/internal/state"
)

func TestPortsUpdateCmd(t *testing.T) {
	// Setup temporary home
	tmpHome, err := os.MkdirTemp("", "git-wt-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Setup fake git project
	projectDir, _ := os.MkdirTemp("", "git-wt-project")
	projectDir = state.NormalizePath(projectDir)
	defer os.RemoveAll(projectDir)
	exec.Command("git", "init", projectDir).Run()
	origWd, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(origWd)

	projectName := filepath.Base(projectDir)
	wt1 := filepath.Join(projectDir, "wt1")
	wt1 = state.NormalizePath(wt1)
	os.MkdirAll(wt1, 0755)

	t.Run("Sync missing ports and remove obsolete ones", func(t *testing.T) {
		// Initial state: wt1 has 'api' port but not 'web'
		state.AppState.Projects = map[string]state.ProjectState{
			projectName: {
				Worktrees: map[string]*state.WorktreeState{
					wt1: {Ports: map[string]int{"api": 0, "old": 5}},
				},
			},
		}
		state.SaveState()

		// New config: 'api' and 'web', but 'old' is gone
		config.AppConfig.Ports = map[string]int{
			"api": 8080,
			"web": 3000,
		}

		t.Logf("Project Name: %s", projectName)
		t.Logf("WT1 Path: %s", wt1)
		t.Logf("State before: %+v", state.AppState.Projects[projectName])

		buf := new(bytes.Buffer)
		portsUpdateCmd.SetOut(buf)

		err := portsUpdateCmd.RunE(portsUpdateCmd, []string{})
		if err != nil {
			t.Fatalf("RunE failed: %v", err)
		}

		t.Logf("State after: %+v", state.AppState.Projects[projectName])

		// Verify state
		updatedState := state.AppState.Projects[projectName].Worktrees[wt1]
		if _, ok := updatedState.Ports["web"]; !ok {
			t.Error("Expected 'web' port to be added")
		}
		if _, ok := updatedState.Ports["old"]; ok {
			t.Error("Expected 'old' port to be removed")
		}
		if updatedState.Ports["api"] != 0 {
			t.Errorf("Expected 'api' port index to remain 0, got %d", updatedState.Ports["api"])
		}

		if _, ok := updatedState.Ports["web"]; ok {
			// web should get smallest available index in this project
			// used: api(0)
			// expected: web(1)
			if updatedState.Ports["web"] != 1 {
				t.Errorf("Expected 'web' port index 1, got %d", updatedState.Ports["web"])
			}
		}
	})
}
