package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssignPortIndex(t *testing.T) {
	AppState.Projects = make(map[string]ProjectState)

	// Assign first index
	idx0 := AssignPortIndex("pj1", "/path/to/wt0", "http", 3000)
	if idx0 != 0 {
		t.Errorf("Expected index 0, got %d", idx0)
	}

	// Assign second index (different worktree, same project)
	idx1 := AssignPortIndex("pj1", "/path/to/wt1", "http", 3000)
	if idx1 != 1 {
		t.Errorf("Expected index 1, got %d", idx1)
	}

	// Assign different port key
	idx_db := AssignPortIndex("pj1", "/path/to/wt0", "db", 9000)
	if idx_db != 0 {
		t.Errorf("Expected index 0 for different key, got %d", idx_db)
	}

	// Re-assign existing path
	idx0_again := AssignPortIndex("pj1", "/path/to/wt0", "http", 3000)
	if idx0_again != 0 {
		t.Errorf("Expected index 0, got %d", idx0_again)
	}

	// The implementation of AssignPortIndex checks usedIndexes across ALL projects for the same port key.
	idx_pj2 := AssignPortIndex("pj2", "/path/to/wt2", "http", 3000)
	if idx_pj2 != 2 {
		t.Errorf("Expected index 2 for project 2 to avoid collision with PJ1, got %d", idx_pj2)
	}
}

func TestCleanupState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-wt-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	existingPath := filepath.Join(tmpDir, "exists")
	os.MkdirAll(existingPath, 0755)
	nonExistingPath := filepath.Join(tmpDir, "missing")

	AppState.Projects = map[string]ProjectState{
		"pj": {
			Worktrees: map[string]WorktreeState{
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

	tmpDir, err := os.MkdirTemp("", "git-wt-match-test")
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
			Worktrees: map[string]WorktreeState{
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
	tmpHome, err := os.MkdirTemp("", "git-wt-home-test")
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
			Worktrees: map[string]WorktreeState{
				"/abs/path/1": {Ports: map[string]int{"http": 10}},
			},
		},
	}

	if err := SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Clear and load
	AppState.Projects = nil
	if err := LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if AppState.Projects["pj1"].Worktrees["/abs/path/1"].Ports["http"] != 10 {
		t.Errorf("Loaded state mismatch: %v", AppState.Projects)
	}
}
