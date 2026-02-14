package state

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func NormalizePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return eval
}

type State struct {
	Projects map[string]ProjectState `json:"projects"` // map[projectName]ProjectState
}

type ProjectState struct {
	Worktrees map[string]*WorktreeState `json:"worktrees"` // map[absPath]*WorktreeState
}

type WorktreeState struct {
	Ports map[string]int `json:"ports"` // map[portKey]portIndex
}

var AppState State

func getStateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zgt", "state.json"), nil
}

func LoadState() error {
	path, err := getStateFilePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		AppState.Projects = make(map[string]ProjectState)
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &AppState)
	if err != nil {
		// If unmarshal fails (e.g. old format), reset state
		AppState.Projects = make(map[string]ProjectState)
		return nil
	}
	if AppState.Projects == nil {
		AppState.Projects = make(map[string]ProjectState)
	}
	return nil
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

func AssignPortIndex(projectName, path, portKey string, basePort int) int {
	path = NormalizePath(path)
	if AppState.Projects == nil {
		AppState.Projects = make(map[string]ProjectState)
	}

	proj, ok := AppState.Projects[projectName]
	if !ok {
		proj = ProjectState{
			Worktrees: make(map[string]*WorktreeState),
		}
	}

	wt, ok := proj.Worktrees[path]
	if !ok {
		wt = &WorktreeState{Ports: make(map[string]int)}
	}

	if idx, ok := wt.Ports[portKey]; ok {
		return idx
	}

	// Determine if this is the main project path (Git root)
	// In the main worktree, .git is a directory.
	// In linked worktrees, .git is a file.
	isMain := false
	gitPath := filepath.Join(path, ".git")
	if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
		isMain = true
	}

	// Find the smallest available index within THIS project
	usedInProject := make(map[int]bool)
	for _, wState := range proj.Worktrees {
		if idx, found := wState.Ports[portKey]; found {
			usedInProject[idx] = true
		}
	}

	idx := 1
	if isMain {
		idx = 0
	}

	for {
		if !usedInProject[idx] {
			if isPortAvailable(basePort + idx) {
				break
			}
			usedInProject[idx] = true
		}
		idx++
	}

	wt.Ports[portKey] = idx
	proj.Worktrees[path] = wt
	AppState.Projects[projectName] = proj
	return idx
}

func isPortAvailable(port int) bool {
	// Simple check using net.Listen
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func ReleasePortIndex(projectName, path string) {
	path = NormalizePath(path)
	if proj, ok := AppState.Projects[projectName]; ok {
		delete(proj.Worktrees, path)
		if len(proj.Worktrees) == 0 {
			delete(AppState.Projects, projectName)
		} else {
			AppState.Projects[projectName] = proj
		}
	}
}

func CleanupState() {
	// Remove paths that no longer exist
	for projectName, proj := range AppState.Projects {
		changed := false
		for path := range proj.Worktrees {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				delete(proj.Worktrees, path)
				changed = true
			}
		}
		if changed {
			if len(proj.Worktrees) == 0 {
				delete(AppState.Projects, projectName)
			} else {
				AppState.Projects[projectName] = proj
			}
		}
	}
}

func GetCurrentWorktreePorts() (map[string]int, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, false
	}
	absCwd := NormalizePath(cwd)

	// Search across all projects for the best matching worktree path
	type wtMatch struct {
		ports map[string]int
		path  string
	}
	var matches []wtMatch

	for _, proj := range AppState.Projects {
		for p, wt := range proj.Worktrees {
			rel, err := filepath.Rel(p, absCwd)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				matches = append(matches, wtMatch{wt.Ports, p})
			}
		}
	}

	if len(matches) == 0 {
		return nil, false
	}

	// Sort by path length descending to get the most specific match
	sort.Slice(matches, func(i, j int) bool {
		return len(matches[i].path) > len(matches[j].path)
	})

	return matches[0].ports, true
}
