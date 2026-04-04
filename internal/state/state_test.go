package state

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAssignPortIndex(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "zgt-assign-test")
	defer os.RemoveAll(tmpDir)

	// Initialize git in tmpDir to make it a "main" project
	exec.Command("git", "init", tmpDir).Run()
	absMain, _ := filepath.Abs(tmpDir)
	projectName := filepath.Base(absMain)

	AppState.Projects = make(map[string]ProjectState)

	// Assign to main (should get 0)
	idx_main := AssignPortIndex(projectName, absMain, "http", 3000)
	if idx_main != 0 {
		t.Errorf("Expected index 0 for main project, got %d", idx_main)
	}

	// Assign to a "worktree" (not the main root)
	wtPath := filepath.Join(tmpDir, "wt0")
	os.MkdirAll(wtPath, 0755)
	idx_wt := AssignPortIndex(projectName, wtPath, "http", 3000)
	if idx_wt != 1 {
		t.Errorf("Expected index 1 for worktree, got %d", idx_wt)
	}

	// Assign another worktree
	wtPath2 := filepath.Join(tmpDir, "wt1")
	os.MkdirAll(wtPath2, 0755)
	idx_wt2 := AssignPortIndex(projectName, wtPath2, "http", 3000)
	if idx_wt2 != 2 {
		t.Errorf("Expected index 2 for second worktree, got %d", idx_wt2)
	}

	// Re-assign existing path
	idx_wt_again := AssignPortIndex(projectName, wtPath, "http", 3000)
	if idx_wt_again != 1 {
		t.Errorf("Expected index 1 for same worktree, got %d", idx_wt_again)
	}

	// Different project should be able to use index 0 as well (project-scoped)
	projectName2 := "other-project"
	idx_pj2 := AssignPortIndex(projectName2, "/other/path", "http", 3000)
	// it starts at 1 for non-main path
	if idx_pj2 != 1 {
		t.Errorf("Expected index 1 for non-main path in other project, got %d", idx_pj2)
	}
}

func TestCleanupState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zgt-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	existingPath := filepath.Join(tmpDir, "exists")
	os.MkdirAll(existingPath, 0755)
	nonExistingPath := filepath.Join(tmpDir, "missing")

	AppState.Projects = map[string]ProjectState{
		"pj": {
			Worktrees: map[string]*WorktreeState{
				existingPath:    {Ports: map[string]int{"http": 0}},
				nonExistingPath: {Ports: map[string]int{"http": 1}},
			},
		},
	}

	CleanupState()

	if _, ok := AppState.Projects["pj"].Worktrees[existingPath]; !ok {
		t.Error("Existing path was incorrectly removed")
	}
	if _, ok := AppState.Projects["pj"].Worktrees[nonExistingPath]; ok {
		t.Error("Non-existing path was not removed")
	}
}

func TestGetCurrentWorktreePorts(t *testing.T) {
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	tmpDir, err := os.MkdirTemp("", "zgt-match-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	wtPath := filepath.Join(tmpDir, "worktree")
	subDir := filepath.Join(wtPath, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	wtPath, _ = filepath.EvalSymlinks(wtPath)
	subDir, _ = filepath.EvalSymlinks(subDir)

	AppState.Projects = map[string]ProjectState{
		"pj": {
			Worktrees: map[string]*WorktreeState{
				wtPath: {Ports: map[string]int{"http": 5}},
			},
		},
	}

	// Test exact match
	if err := os.Chdir(wtPath); err != nil {
		t.Fatal(err)
	}
	ports, found := GetCurrentWorktreePorts()
	if !found || ports["http"] != 5 {
		t.Errorf("Expected {http:5}, got %v", ports)
	}

	// Test subdirectory match
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	ports, found = GetCurrentWorktreePorts()
	if !found || ports["http"] != 5 {
		t.Errorf("Expected {http:5} for subdirectory, got %v", ports)
	}
}

func TestSaveLoadState(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "zgt-home-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Setup state
	AppState.Projects = map[string]ProjectState{
		"pj1": {
			Worktrees: map[string]*WorktreeState{
				"/abs/path/1": {Ports: map[string]int{"http": 10}},
			},
		},
	}
	AppState.TmuxSessions = map[string]TmuxSessionState{
		"ses1": {
			Windows: []TmuxWindowSaveState{
				{Name: "win1", CWD: "/path/to/win1", IsZgt: true, ZgtWorktree: "feat1"},
				{Name: "win2", CWD: "/path/to/win2", IsZgt: false},
			},
		},
	}

	if err := SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Clear and load
	AppState.Projects = nil
	AppState.TmuxSessions = nil
	if err := LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if AppState.Projects["pj1"].Worktrees["/abs/path/1"].Ports["http"] != 10 {
		t.Errorf("Loaded project state mismatch: %v", AppState.Projects)
	}

	ses, ok := AppState.TmuxSessions["ses1"]
	if !ok || len(ses.Windows) != 2 {
		t.Errorf("Loaded tmux sessions mismatch: %v", AppState.TmuxSessions)
	}
	if ses.Windows[0].Name != "win1" || !ses.Windows[0].IsZgt || ses.Windows[0].ZgtWorktree != "feat1" {
		t.Errorf("Loaded window 0 mismatch: %v", ses.Windows[0])
	}
	if ses.Windows[1].Name != "win2" || ses.Windows[1].IsZgt {
		t.Errorf("Loaded window 1 mismatch: %v", ses.Windows[1])
	}
}
