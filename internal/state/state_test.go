package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssignPortIndex(t *testing.T) {
	AppState.Worktrees = make(map[string]int)

	// Assign first index
	idx0 := AssignPortIndex("/path/to/wt0")
	if idx0 != 0 {
		t.Errorf("Expected index 0, got %d", idx0)
	}

	// Assign second index
	idx1 := AssignPortIndex("/path/to/wt1")
	if idx1 != 1 {
		t.Errorf("Expected index 1, got %d", idx1)
	}

	// Re-assign existing path
	idx0_again := AssignPortIndex("/path/to/wt0")
	if idx0_again != 0 {
		t.Errorf("Expected index 0, got %d", idx0_again)
	}

	// Release 0 and assign new (should get 0)
	ReleasePortIndex("/path/to/wt0")
	idx0_recycled := AssignPortIndex("/path/to/wt2")
	if idx0_recycled != 0 {
		t.Errorf("Expected recycled index 0, got %d", idx0_recycled)
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

	AppState.Worktrees = map[string]int{
		existingPath:    0,
		nonExistingPath: 1,
	}

	CleanupState()

	if _, ok := AppState.Worktrees[existingPath]; !ok {
		t.Error("Existing path was incorrectly removed")
	}
	if _, ok := AppState.Worktrees[nonExistingPath]; ok {
		t.Error("Non-existing path was not removed")
	}
}

func TestGetCurrentWorktreePortIndex(t *testing.T) {
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

	AppState.Worktrees = map[string]int{
		wtPath: 5,
	}

	// Test exact match
	if err := os.Chdir(wtPath); err != nil {
		t.Fatal(err)
	}
	idx, found := GetCurrentWorktreePortIndex()
	if !found || idx != 5 {
		t.Errorf("Expected (5, true), got (%d, %t) for exact match", idx, found)
	}

	// Test subdirectory match
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	idx, found = GetCurrentWorktreePortIndex()
	if !found || idx != 5 {
		t.Errorf("Expected (5, true), got (%d, %t) for subdirectory", idx, found)
	}

	// Test no match (parent directory)
	os.Chdir(tmpDir)
	idx, found = GetCurrentWorktreePortIndex()
	if found {
		t.Errorf("Expected found:false for parent directory, got true with idx %d", idx)
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
	AppState.Worktrees = map[string]int{
		"/abs/path/1": 10,
		"/abs/path/2": 20,
	}

	if err := SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Clear and load
	AppState.Worktrees = nil
	if err := LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if AppState.Worktrees["/abs/path/1"] != 10 || AppState.Worktrees["/abs/path/2"] != 20 {
		t.Errorf("Loaded state mismatch: %v", AppState.Worktrees)
	}
}
