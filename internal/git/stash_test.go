package git

import (
	"os"
	"os/exec"
	"testing"
)

func TestStashHelpers(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "zgt-stash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to tmpDir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Init a real git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user for stash to work in CI
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "Test User").Run()

	t.Run("HasUncommittedFiles", func(t *testing.T) {
		uncommitted, err := HasUncommittedFiles()
		if err != nil {
			t.Fatalf("HasUncommittedFiles failed: %v", err)
		}
		if uncommitted {
			t.Error("expected false for empty repo, got true")
		}

		// Create a file
		if err := os.WriteFile("test.txt", []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		uncommitted, err = HasUncommittedFiles()
		if err != nil {
			t.Fatalf("HasUncommittedFiles failed: %v", err)
		}
		if !uncommitted {
			t.Error("expected true after creating untracked file, got false")
		}

		// Commit it
		exec.Command("git", "add", "test.txt").Run()
		exec.Command("git", "commit", "-m", "initial").Run()

		uncommitted, err = HasUncommittedFiles()
		if err != nil {
			t.Fatalf("HasUncommittedFiles failed: %v", err)
		}
		if uncommitted {
			t.Error("expected false after commit, got true")
		}

		// Modify it
		if err := os.WriteFile("test.txt", []byte("world"), 0644); err != nil {
			t.Fatal(err)
		}

		uncommitted, err = HasUncommittedFiles()
		if err != nil {
			t.Fatalf("HasUncommittedFiles failed: %v", err)
		}
		if !uncommitted {
			t.Error("expected true after modifying file, got false")
		}
	})

	t.Run("StashAndPop", func(t *testing.T) {
		// Repo has modified test.txt from previous test
		err := Stash("test stash")
		if err != nil {
			t.Fatalf("Stash failed: %v", err)
		}

		uncommitted, _ := HasUncommittedFiles()
		if uncommitted {
			t.Error("repo should be clean after stash")
		}

		err = StashPop()
		if err != nil {
			t.Fatalf("StashPop failed: %v", err)
		}

		uncommitted, _ = HasUncommittedFiles()
		if !uncommitted {
			t.Error("repo should have uncommitted changes after stash pop")
		}
	})
}
