package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveHooksPath(t *testing.T) {
	t.Run("relative path uses project root", func(t *testing.T) {
		got, err := ResolveHooksPath("/tmp/project", ".githooks")
		if err != nil {
			t.Fatalf("ResolveHooksPath returned error: %v", err)
		}
		want := filepath.Join("/tmp/project", ".githooks")
		if got != want {
			t.Fatalf("ResolveHooksPath() = %s, want %s", got, want)
		}
	})

	t.Run("absolute path is preserved", func(t *testing.T) {
		got, err := ResolveHooksPath("/tmp/project", "/opt/hooks")
		if err != nil {
			t.Fatalf("ResolveHooksPath returned error: %v", err)
		}
		if got != "/opt/hooks" {
			t.Fatalf("ResolveHooksPath() = %s, want /opt/hooks", got)
		}
	})
}

func TestRegisterHooksPath(t *testing.T) {
	t.Run("registers hooks path in worktree config", func(t *testing.T) {
		_, worktreePath, hooksPath := setupWorktreeRepo(t)
		registered, existingPath, err := RegisterHooksPath(worktreePath, hooksPath)
		if err != nil {
			t.Fatalf("RegisterHooksPath returned error: %v", err)
		}
		if !registered {
			t.Fatalf("expected registration to succeed, existingPath=%q", existingPath)
		}

		got := gitOutput(t, worktreePath, "config", "--get", "core.hooksPath")
		want, _ := filepath.Abs(hooksPath)
		if got != want {
			t.Fatalf("core.hooksPath = %s, want %s", got, want)
		}
	})

	t.Run("skips when hooks path already exists", func(t *testing.T) {
		repoRoot, worktreePath, hooksPath := setupWorktreeRepo(t)
		existing := filepath.Join(repoRoot, ".existing-hooks")
		if err := os.MkdirAll(existing, 0755); err != nil {
			t.Fatalf("failed to create existing hooks dir: %v", err)
		}
		runGit(t, worktreePath, "config", "extensions.worktreeConfig", "true")
		runGit(t, worktreePath, "config", "--worktree", "core.hooksPath", existing)

		registered, existingPath, err := RegisterHooksPath(worktreePath, hooksPath)
		if err != nil {
			t.Fatalf("RegisterHooksPath returned error: %v", err)
		}
		if registered {
			t.Fatal("expected registration to be skipped")
		}
		if existingPath != existing {
			t.Fatalf("existingPath = %s, want %s", existingPath, existing)
		}

		got := gitOutput(t, worktreePath, "config", "--get", "core.hooksPath")
		if got != existing {
			t.Fatalf("core.hooksPath = %s, want %s", got, existing)
		}
	})

	t.Run("errors when hooks path is missing", func(t *testing.T) {
		repoRoot, worktreePath, _ := setupWorktreeRepo(t)
		missing := filepath.Join(repoRoot, ".missing-hooks")
		registered, existingPath, err := RegisterHooksPath(worktreePath, missing)
		if err == nil {
			t.Fatal("expected error for missing hooks path")
		}
		if registered {
			t.Fatal("expected registration to fail")
		}
		if existingPath != "" {
			t.Fatalf("existingPath = %q, want empty", existingPath)
		}
	})
}

func setupWorktreeRepo(t *testing.T) (string, string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "repo")
	worktreePath := filepath.Join(tmpDir, "repo-feature")
	hooksPath := filepath.Join(repoRoot, ".githooks")

	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	runGit(t, repoRoot, "init", "-b", "main")
	runGit(t, repoRoot, "config", "user.name", "zgt-test")
	runGit(t, repoRoot, "config", "user.email", "zgt@example.com")

	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "commit", "-m", "init")

	if err := os.MkdirAll(hooksPath, 0755); err != nil {
		t.Fatalf("failed to create hooks path: %v", err)
	}
	runGit(t, repoRoot, "worktree", "add", worktreePath, "-b", "feature")

	return repoRoot, worktreePath, hooksPath
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}
