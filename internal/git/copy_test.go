package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCopyIgnoredFiles(t *testing.T) {
	// Setup temporary source and target directories
	tmpSrc, err := os.MkdirTemp("", "git-wt-copy-test-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpSrc)

	tmpDst, err := os.MkdirTemp("", "git-wt-copy-test-dst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDst)

	// Initialize git repo in source
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpSrc
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping git-dependent test as git init failed")
	}

	// Create a .gitignore file
	gitignoreContent := ".env\nnode_modules/\n"
	err = os.WriteFile(filepath.Join(tmpSrc, ".gitignore"), []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create some ignored files
	envFile := filepath.Join(tmpSrc, ".env")
	os.WriteFile(envFile, []byte("API_KEY=12345"), 0644)

	nmDir := filepath.Join(tmpSrc, "node_modules")
	os.MkdirAll(nmDir, 0755)
	os.WriteFile(filepath.Join(nmDir, "dummy"), []byte("dummy content"), 0644)

	// Create git-wt.config.yml (untracked/uncommitted)
	configFile := filepath.Join(tmpSrc, "git-wt.config.yml")
	configContent := "ports:\n  api: 8080\n"
	os.WriteFile(configFile, []byte(configContent), 0644)

	// Create a committed file (should NOT be copied by CopyIgnoredFiles usually,
	// unless it's explicitly ignored, but CopyIgnoredFiles uses git ls-files --others --ignored)
	committedFile := filepath.Join(tmpSrc, "README.md")
	os.WriteFile(committedFile, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", tmpSrc, "add", "README.md", ".gitignore").Run()
	exec.Command("git", "-C", tmpSrc, "commit", "-m", "initial commit").Run()

	t.Run("Do NOT copy untracked config files if not ignored", func(t *testing.T) {
		err := CopyIgnoredFiles(tmpSrc, tmpDst, []string{}, false)
		if err != nil {
			t.Fatalf("CopyIgnoredFiles failed: %v", err)
		}

		// Verify .env is copied (it's in .gitignore)
		if _, err := os.Stat(filepath.Join(tmpDst, ".env")); os.IsNotExist(err) {
			t.Error(".env should have been copied (ignored)")
		}

		// Verify git-wt.config.yml is NOT copied (it's untracked and NOT in .gitignore)
		if _, err := os.Stat(filepath.Join(tmpDst, "git-wt.config.yml")); err == nil {
			t.Error("git-wt.config.yml should NOT have been copied (untracked and not ignored)")
		}
	})

	t.Run("Do NOT copy untracked config files even IF ignored", func(t *testing.T) {
		// Reset destination
		os.RemoveAll(tmpDst)
		os.MkdirAll(tmpDst, 0755)

		// Add git-wt.config.yml to .gitignore
		f, _ := os.OpenFile(filepath.Join(tmpSrc, ".gitignore"), os.O_APPEND|os.O_WRONLY, 0644)
		f.WriteString("git-wt.config.yml\n")
		f.Close()

		err := CopyIgnoredFiles(tmpSrc, tmpDst, []string{}, false)
		if err != nil {
			t.Fatalf("CopyIgnoredFiles failed: %v", err)
		}

		// Verify git-wt.config.yml is NOT copied
		if _, err := os.Stat(filepath.Join(tmpDst, "git-wt.config.yml")); err == nil {
			t.Error("git-wt.config.yml should NOT have been copied because it is now explicitly excluded")
		}
	})

	t.Run("Respect ignorePatterns", func(t *testing.T) {
		// Reset destination
		os.RemoveAll(tmpDst)
		os.MkdirAll(tmpDst, 0755)

		// Ignore node_modules
		err := CopyIgnoredFiles(tmpSrc, tmpDst, []string{"node_modules/**"}, false)
		if err != nil {
			t.Fatalf("CopyIgnoredFiles failed: %v", err)
		}

		// Verify .env is copied
		if _, err := os.Stat(filepath.Join(tmpDst, ".env")); os.IsNotExist(err) {
			t.Error(".env should still be copied")
		}

		// Verify node_modules/dummy is NOT copied
		if _, err := os.Stat(filepath.Join(tmpDst, "node_modules", "dummy")); err == nil {
			t.Error("node_modules/dummy should NOT have been copied due to ignore patterns")
		}
	})
}
