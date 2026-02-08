package template

import "testing"

func TestReplace(t *testing.T) {
	ctx := Context{
		Path:   "/path/to/repo-branch",
		Branch: "branch",
		Repo:   "repo",
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
			name:     "Replace multiple",
			tmpl:     "path: {{.Path}}, branch: {{.Branch}}, repo: {{.Repo}}",
			expected: "path: /path/to/repo-branch, branch: branch, repo: repo",
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
