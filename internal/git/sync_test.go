package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSyncLogic(t *testing.T) {
	// Setup temporary source and target directories
	tmpSrc, err := os.MkdirTemp("", "zgt-sync-test-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpSrc)

	tmpDst, err := os.MkdirTemp("", "zgt-sync-test-dst")
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
	gitignoreContent := ".env\n"
	err = os.WriteFile(filepath.Join(tmpSrc, ".gitignore"), []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create an ignored file
	envFile := filepath.Join(tmpSrc, ".env")
	envContent := "API_KEY=test-key"
	os.WriteFile(envFile, []byte(envContent), 0644)

	// Create a normal file
	readmeFile := filepath.Join(tmpSrc, "README.md")
	os.WriteFile(readmeFile, []byte("# README"), 0644)

	t.Run("GetIgnoredFiles", func(t *testing.T) {
		files, err := GetIgnoredFiles(tmpSrc)
		if err != nil {
			t.Fatalf("GetIgnoredFiles failed: %v", err)
		}

		found := false
		for _, f := range files {
			if f == ".env" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf(".env should be in ignored files, but got %v", files)
		}

		for _, f := range files {
			if f == "README.md" {
				t.Errorf("README.md should NOT be in ignored files")
			}
		}
	})

	t.Run("SyncFiles", func(t *testing.T) {
		err := SyncFiles(tmpSrc, tmpDst, []string{".env"})
		if err != nil {
			t.Fatalf("SyncFiles failed: %v", err)
		}

		// Verify .env is copied
		dstFile := filepath.Join(tmpDst, ".env")
		content, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Copied file .env not found or unreadable: %v", err)
		}

		if string(content) != envContent {
			t.Errorf("Copied file content mismatch. Got %s, want %s", string(content), envContent)
		}
	})
}
