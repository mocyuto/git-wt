package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type State struct {
	Worktrees map[string]int `json:"worktrees"` // map[absPath]portIndex
}

var AppState State

func getStateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "git-wt", "state.json"), nil
}

func LoadState() error {
	path, err := getStateFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		AppState.Worktrees = make(map[string]int)
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &AppState)
}

func SaveState() error {
	path, err := getStateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(AppState, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func AssignPortIndex(path string) int {
	if idx, ok := AppState.Worktrees[path]; ok {
		return idx
	}

	// Find the smallest available index
	used := make(map[int]bool)
	for _, idx := range AppState.Worktrees {
		used[idx] = true
	}

	idx := 0
	for {
		if !used[idx] {
			break
		}
		idx++
	}

	AppState.Worktrees[path] = idx
	return idx
}

func ReleasePortIndex(path string) {
	delete(AppState.Worktrees, path)
}

func CleanupState() {
	// Remove paths that no longer exist
	for path := range AppState.Worktrees {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(AppState.Worktrees, path)
		}
	}
}

func GetCurrentWorktreePortIndex() (int, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return 0, false
	}
	absCwd, _ := filepath.Abs(cwd)

	// Check if we are in one of the registered worktrees
	// We handle subdirectories by checking if absCwd starts with one of the registered paths

	// Sort registered paths by length (longest first) to match most specific worktree
	type wt struct {
		path string
		idx  int
	}
	var wts []wt
	for p, i := range AppState.Worktrees {
		wts = append(wts, wt{p, i})
	}
	sort.Slice(wts, func(i, j int) bool {
		return len(wts[i].path) > len(wts[j].path)
	})

	for _, w := range wts {
		rel, err := filepath.Rel(w.path, absCwd)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return w.idx, true
		}
	}

	return 0, false
}
