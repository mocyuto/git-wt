package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInitConfig(t *testing.T) {
	// Setup temporary home and project root
	tmpHome, err := os.MkdirTemp("", "git-wt-config-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	tmpGit, err := os.MkdirTemp("", "git-wt-config-test-git")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpGit)

	// Initialize git repo to satisfy git.GetGitRoot()
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpGit
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping git-dependent test as git init failed")
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	origWd, _ := os.Getwd()
	os.Chdir(tmpGit)
	defer os.Chdir(origWd)

	t.Run("Default global config creation", func(t *testing.T) {
		AppConfig = Config{} // Reset
		InitConfig()

		configPath := filepath.Join(tmpHome, ".config", "git-wt", "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Global config.yaml should have been created with defaults")
		}
	})

	t.Run("Global config loading", func(t *testing.T) {
		configDir := filepath.Join(tmpHome, ".config", "git-wt")
		os.MkdirAll(configDir, 0755)
		os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(`
hooks:
  add: ["global-add"]
ignore: [".global-ignore"]
`), 0644)

		AppConfig = Config{} // Reset
		InitConfig()

		if len(AppConfig.Hooks.Add) != 1 || AppConfig.Hooks.Add[0] != "global-add" {
			t.Errorf("Expected global hook 'global-add', got %v", AppConfig.Hooks.Add)
		}
		if len(AppConfig.Ignore) != 1 || AppConfig.Ignore[0] != ".global-ignore" {
			t.Errorf("Expected global ignore '.global-ignore', got %v", AppConfig.Ignore)
		}
	})

	t.Run("Local config merging", func(t *testing.T) {
		// Local config in tmpGit (current working directory)
		os.WriteFile(filepath.Join(tmpGit, "git-wt.config.yaml"), []byte(`
hooks:
  add: ["local-add"]
ignore: [".local-ignore"]
`), 0644)

		// Still have previous global config in tmpHome
		AppConfig = Config{} // Reset
		InitConfig()

		// Hooks should be merged
		foundGlobal := false
		foundLocal := false
		for _, h := range AppConfig.Hooks.Add {
			if h == "global-add" {
				foundGlobal = true
			}
			if h == "local-add" {
				foundLocal = true
			}
		}
		if !foundGlobal || !foundLocal {
			t.Errorf("Hooks not merged correctly: %v", AppConfig.Hooks.Add)
		}

		// Ignores should be merged
		if len(AppConfig.Ignore) != 2 {
			t.Errorf("Ignores not merged correctly: %v", AppConfig.Ignore)
		}
	})

	t.Run("Explicit config override", func(t *testing.T) {
		explicitPath := filepath.Join(tmpGit, "explicit.yaml")
		os.WriteFile(explicitPath, []byte(`
ports:
  web: 3000
`), 0644)

		AppConfig = Config{} // Reset
		CfgFile = explicitPath
		defer func() { CfgFile = "" }()

		InitConfig()

		if AppConfig.Ports["web"] != 3000 {
			t.Errorf("Expected explicit port 3000, got %v", AppConfig.Ports["web"])
		}
		// Explicit config usually overrides global/local in this implementation (step 3 in InitConfig)
		// but check how it's implemented. Step 3 is an unmarshal into AppConfig which might fully override
	})
}
