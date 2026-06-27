package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/mocyuto/zgt/internal/state"
	"github.com/mocyuto/zgt/internal/template"
)

func TestRunHooks_Env(t *testing.T) {
	// Setup config
	config.AppConfig.Env = map[string]string{
		"TEST_BRANCH": "{{.Branch}}",
		"TEST_STATIC": "hello",
	}
	config.AppConfig.Hooks.Add = []string{
		"echo $TEST_BRANCH > branch_out.txt",
		"echo $TEST_STATIC > static_out.txt",
	}

	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	ctx := template.Context{
		Path:   "test-path",
		Branch: "feature-xyz",
		Repo:   "test-repo",
	}

	RunHooks("add", ctx)

	// Verify outputs
	branchOut, err := os.ReadFile("branch_out.txt")
	if err != nil {
		t.Fatalf("Failed to read branch_out.txt: %v", err)
	}
	if strings.TrimSpace(string(branchOut)) != "feature-xyz" {
		t.Errorf("Expected branch_out.txt to be 'feature-xyz', got '%s'", strings.TrimSpace(string(branchOut)))
	}

	staticOut, err := os.ReadFile("static_out.txt")
	if err != nil {
		t.Fatalf("Failed to read static_out.txt: %v", err)
	}
	if strings.TrimSpace(string(staticOut)) != "hello" {
		t.Errorf("Expected static_out.txt to be 'hello', got '%s'", strings.TrimSpace(string(staticOut)))
	}
}

func TestRunHooks_Profile(t *testing.T) {
	// Reset config between tests.
	config.AppConfig = config.Config{}

	config.AppConfig.Env = map[string]string{
		"SHARED_VAR": "base",
	}
	config.AppConfig.Hooks.Add = []string{
		"echo base:$SHARED_VAR:$PROFILE_VAR > base_out.txt",
	}

	// migration profile overrides SHARED_VAR and supplies PROFILE_VAR +
	// appends an extra hook.
	config.AppConfig.Profiles = map[string]config.ProfileConfig{
		"migration": {
			Env: map[string]string{
				"SHARED_VAR":  "overridden",
				"PROFILE_VAR": "iso",
			},
			Hooks: config.HooksConfig{
				Add: []string{"echo extra:$SHARED_VAR > extra_out.txt"},
			},
		},
	}

	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	t.Run("default profile", func(t *testing.T) {
		os.Remove("base_out.txt")
		os.Remove("extra_out.txt")
		ctx := template.Context{
			Path:   "test-path",
			Branch: "feat",
			Repo:   "repo",
		}
		RunHooks("add", ctx)

		out, err := os.ReadFile("base_out.txt")
		if err != nil {
			t.Fatalf("base_out.txt: %v", err)
		}
		// PROFILE_VAR is unset in default profile -> empty string after shell
		if got := strings.TrimSpace(string(out)); got != "base:base:" {
			t.Errorf("default base hook output = %q, want %q", got, "base:base:")
		}
		if _, err := os.Stat("extra_out.txt"); !os.IsNotExist(err) {
			t.Errorf("extra hook from migration profile should NOT run under default profile")
		}
	})

	t.Run("migration profile", func(t *testing.T) {
		os.Remove("base_out.txt")
		os.Remove("extra_out.txt")
		ctx := template.Context{
			Path:    "test-path",
			Branch:  "feat",
			Repo:    "repo",
			Profile: "migration",
		}
		RunHooks("add", ctx)

		out, err := os.ReadFile("base_out.txt")
		if err != nil {
			t.Fatalf("base_out.txt: %v", err)
		}
		if got := strings.TrimSpace(string(out)); got != "base:overridden:iso" {
			t.Errorf("migration base hook output = %q, want %q", got, "base:overridden:iso")
		}

		extra, err := os.ReadFile("extra_out.txt")
		if err != nil {
			t.Fatalf("extra_out.txt: %v", err)
		}
		// PROFILE_VAR is also visible to the appended profile-specific hook.
		if got := strings.TrimSpace(string(extra)); got != "extra:overridden" {
			t.Errorf("migration extra hook output = %q, want %q", got, "extra:overridden")
		}
	})
}

// TestRunHooks_PortEnvInjection verifies that hook subprocesses receive
// NAME_PORT env vars matching the worktree's port assignment in state, so
// compose-style hooks that interpolate ${API_PORT} pick up the worktree's
// assigned port without a prior `eval "$(zgt env)"`.
func TestRunHooks_PortEnvInjection(t *testing.T) {
	config.AppConfig = config.Config{
		Ports: map[string]int{"api": 3000, "envoy": 8080},
		Hooks: config.HooksConfig{
			Add: []string{"echo $API_PORT,$ENVOY_PORT > port_out.txt"},
		},
	}

	tmpRepo := t.TempDir()
	worktreePath := filepath.Join(tmpRepo, "wt")
	os.MkdirAll(worktreePath, 0755)

	// Seed state with port indices for this worktree path.
	state.AppState.Projects = map[string]state.ProjectState{
		"pj": {
			Worktrees: map[string]*state.WorktreeState{
				state.NormalizePath(worktreePath): {
					Ports: map[string]int{"api": 4, "envoy": 4},
				},
			},
		},
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpRepo)
	defer os.Chdir(origDir)

	ctx := template.Context{Path: worktreePath, Branch: "feat", Repo: "pj"}
	RunHooks("add", ctx)

	out, err := os.ReadFile("port_out.txt")
	if err != nil {
		t.Fatalf("port_out.txt: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "3004,8084" {
		t.Errorf("hook saw ports %q, want %q", got, "3004,8084")
	}
}
