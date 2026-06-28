package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mocyuto/zgt/internal/git"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/stretchr/testify/assert"
)

func TestTmuxOpenCmd(t *testing.T) {
	// Test that the command is correctly initialized and has the right properties
	assert.Equal(t, "open [worktree-name]", tmuxOpenCmd.Use)
	assert.NotEmpty(t, tmuxOpenCmd.Short)
}

func TestTmuxOpenCmdHasProfileFlag(t *testing.T) {
	flag := tmuxOpenCmd.Flags().Lookup("profile")
	assert.NotNil(t, flag, "tmux open command should have a --profile flag")
	if flag != nil {
		assert.Equal(t, "profile", flag.Name)
		assert.Equal(t, "", flag.DefValue, "default value should be empty")
		assert.NotEmpty(t, flag.Usage, "usage string should be set")
	}
}

func TestResolveProfilesUsesStateWhenNoOverride(t *testing.T) {
	// Set up a temporary git repo so gitroot.GetMainProjectRoot succeeds,
	// and set cwd inside it so it's resolved as the main project.
	tmpDir, err := os.MkdirTemp("", "zgt-tmux-open-resolve-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := exec.Command("git", "init", tmpDir).Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to cd: %v", err)
	}

	// Isolate state file by redirecting HOME to tmpDir; LoadState / SaveState
	// will then resolve to tmpDir/.config/zgt/state.json.
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	projectName := filepath.Base(tmpDir)
	wtPath := filepath.Join(tmpDir, "wt0")
	if err := os.MkdirAll(wtPath, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Seed in-memory state with a known profile. resolveProfiles will reload
	// from disk, so persist via the state package itself to keep state.json
	// authoritative for this test.
	state.AppState.Projects = map[string]state.ProjectState{
		projectName: {
			Worktrees: map[string]*state.WorktreeState{
				state.NormalizePath(wtPath): {Profile: "alpha"},
			},
		},
	}
	if err := state.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}
	defer func() { state.AppState.Projects = nil }()

	profiles, err := resolveProfiles([]git.Worktree{{Path: wtPath}}, "")
	assert.NoError(t, err)
	assert.Equal(t, "alpha", profiles[state.NormalizePath(wtPath)])
}

func TestTmuxOpenRejectsUnknownProfile(t *testing.T) {
	old := tmuxOpenProfileFlag
	defer func() { tmuxOpenProfileFlag = old }()
	tmuxOpenProfileFlag = "does-not-exist"

	err := tmuxOpenCmd.RunE(tmuxOpenCmd, []string{""})
	assert.Error(t, err, "expected error for unknown profile")
}

func TestResolveProfilesPersistsOverride(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zgt-tmux-open-override-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := exec.Command("git", "init", tmpDir).Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to cd: %v", err)
	}
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	projectName := filepath.Base(tmpDir)
	wtPath := filepath.Join(tmpDir, "wt0")
	if err := os.MkdirAll(wtPath, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	defer func() { state.AppState.Projects = nil }()

	profiles, err := resolveProfiles([]git.Worktree{{Path: wtPath}}, "beta")
	assert.NoError(t, err)
	absWt := state.NormalizePath(wtPath)
	assert.Equal(t, "beta", profiles[absWt], "returned profile should match override")

	// Re-read from disk via a fresh LoadState cycle to confirm persistence.
	state.AppState.Projects = nil
	if err := state.LoadState(); err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	got := state.AppState.Projects[projectName].Worktrees[absWt].Profile
	assert.Equal(t, "beta", got, "override should have been persisted for the worktree")
}
