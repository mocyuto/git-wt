package template

import (
	"reflect"
	"testing"

	"github.com/mocyuto/zgt/internal/config"
)

func TestReplace(t *testing.T) {
	ctx := Context{
		Path:          "/path/to/repo-branch",
		Branch:        "branch",
		Repo:          "repo",
		CurrentDir:    "curr-dir",
		TargetBranch:  "target",
		CurrentBranch: "curr-branch",
	}

	tests := []struct {
		name     string
		tmpl     string
		expected string
	}{
		{
			name:     "Replace Path",
			tmpl:     "echo {{.Path}}",
			expected: "echo /path/to/repo-branch",
		},
		{
			name:     "Replace Branch",
			tmpl:     "checkout {{.Branch}}",
			expected: "checkout branch",
		},
		{
			name:     "Replace Repo",
			tmpl:     "project {{.Repo}}",
			expected: "project repo",
		},
		{
			name:     "Replace CurrentDir",
			tmpl:     "dir {{.CurrentDir}}",
			expected: "dir curr-dir",
		},
		{
			name:     "Replace TargetBranch",
			tmpl:     "target {{.TargetBranch}}",
			expected: "target target",
		},
		{
			name:     "Replace CurrentBranch",
			tmpl:     "current {{.CurrentBranch}}",
			expected: "current curr-branch",
		},
		{
			name:     "Replace multiple",
			tmpl:     "path: {{.Path}}, branch: {{.Branch}}, repo: {{.Repo}}, dir: {{.CurrentDir}}, target: {{.TargetBranch}}, current: {{.CurrentBranch}}",
			expected: "path: /path/to/repo-branch, branch: branch, repo: repo, dir: curr-dir, target: target, current: curr-branch",
		},
		{
			name:     "No placeholders",
			tmpl:     "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Replace(tt.tmpl, ctx)
			if got != tt.expected {
				t.Errorf("Replace() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReplaceMap(t *testing.T) {
	ctx := Context{
		Path:          "/path/to/repo-branch",
		Branch:        "branch",
		Repo:          "repo",
		CurrentDir:    "curr-dir",
		TargetBranch:  "target",
		CurrentBranch: "curr-branch",
	}

	m := map[string]string{
		"B": "{{.Branch}}",
		"P": "{{.Path}}",
		"R": "{{.Repo}}",
		"D": "{{.CurrentDir}}",
		"T": "{{.TargetBranch}}",
		"C": "{{.CurrentBranch}}",
		"S": "static",
	}

	got := ReplaceMap(m, ctx)

	expected := map[string]string{
		"B": "branch",
		"P": "/path/to/repo-branch",
		"R": "repo",
		"D": "curr-dir",
		"T": "target",
		"C": "curr-branch",
		"S": "static",
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ReplaceMap() = %v, want %v", got, expected)
	}
}

func TestReplaceConfig(t *testing.T) {
	ctx := Context{
		Path:          "/path",
		Branch:        "feat",
		Repo:          "myrepo",
		CurrentDir:    "myrepo-feat",
		TargetBranch:  "feat",
		CurrentBranch: "main",
	}

	cfg := config.Config{}
	cfg.Hooks.Add = []string{"echo {{.Branch}}"}
	cfg.Hooks.RM = []string{"rm {{.Path}}"}
	cfg.Ignore = []string{"{{.Repo}}.log"}
	cfg.Env = map[string]string{"G": "{{.Repo}}"}
	cfg.Tmux.WindowName = "work-{{.Branch}}"
	cfg.Tmux.Panes = []config.TmuxPane{{Commands: []string{"echo {{.Repo}}"}}}

	got := ReplaceConfig(cfg, ctx)

	if got.Hooks.Add[0] != "echo feat" {
		t.Errorf("Hooks.Add mismatch: %v", got.Hooks.Add)
	}
	if got.Hooks.RM[0] != "rm /path" {
		t.Errorf("Hooks.RM mismatch: %v", got.Hooks.RM)
	}
	if got.Ignore[0] != "myrepo.log" {
		t.Errorf("Ignore mismatch: %v", got.Ignore)
	}
	if got.Env["G"] != "myrepo" {
		t.Errorf("Env mismatch: %v", got.Env)
	}
	if got.Tmux.WindowName != "work-feat" {
		t.Errorf("Tmux.WindowName mismatch: %v", got.Tmux.WindowName)
	}
	if got.Tmux.Panes[0].Commands[0] != "echo myrepo" {
		t.Errorf("Tmux.Panes[0].Commands[0] mismatch: %v", got.Tmux.Panes[0].Commands[0])
	}
}
