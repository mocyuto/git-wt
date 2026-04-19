package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInitConfig(t *testing.T) {
	// Setup temporary home and project root
	tmpHome, err := os.MkdirTemp("", "zgt-config-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	tmpGit, err := os.MkdirTemp("", "zgt-config-test-git")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpGit)

	// Initialize git repo to satisfy gitroot.GetGitRoot()
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

		configPath, _ := GetGlobalConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("%s should have been created with defaults", configPath)
		}

		if AppConfig.Tmux.KeepOpen {
			t.Errorf("Expected default Tmux.KeepOpen to be false")
		}
		if AppConfig.GitHooks.Path != ".githooks" {
			t.Errorf("Expected default GitHooks.Path to be .githooks, got %q", AppConfig.GitHooks.Path)
		}
	})

	t.Run("Global config loading", func(t *testing.T) {
		configPath, _ := GetGlobalConfigPath()
		configDir := filepath.Dir(configPath)
		os.MkdirAll(configDir, 0755)
		os.WriteFile(configPath, []byte(`
hooks:
  add: ["global-add"]
ignore: [".global-ignore"]
git_hooks:
  enabled: true
`), 0644)

		AppConfig = Config{} // Reset
		InitConfig()

		if len(AppConfig.Hooks.Add) != 1 || AppConfig.Hooks.Add[0] != "global-add" {
			t.Errorf("Expected global hook 'global-add', got %v", AppConfig.Hooks.Add)
		}
		if len(AppConfig.Ignore) != 1 || AppConfig.Ignore[0] != ".global-ignore" {
			t.Errorf("Expected global ignore '.global-ignore', got %v", AppConfig.Ignore)
		}
		if !AppConfig.GitHooks.Enabled {
			t.Errorf("Expected global git hooks to be enabled")
		}
	})

	t.Run("Legacy githooks alias loading", func(t *testing.T) {
		configPath, _ := GetGlobalConfigPath()
		configDir := filepath.Dir(configPath)
		os.MkdirAll(configDir, 0755)
		os.WriteFile(configPath, []byte(`
hooks:
  add: ["global-add"]
ignore: [".global-ignore"]
githooks:
  enabled: true
  path: .legacy-hooks
`), 0644)

		for _, name := range LocalConfigNames {
			os.Remove(filepath.Join(tmpGit, name))
		}

		AppConfig = Config{}
		InitConfig()

		if !AppConfig.GitHooks.Enabled {
			t.Errorf("Expected legacy githooks alias to enable git hooks")
		}
		if AppConfig.GitHooks.Path != ".legacy-hooks" {
			t.Errorf("Expected legacy githooks path '.legacy-hooks', got %q", AppConfig.GitHooks.Path)
		}
	})

	t.Run("Local config merging (git-wt.config.yaml)", func(t *testing.T) {
		// Remove any existing local config first to ensure clean state
		for _, name := range LocalConfigNames {
			os.Remove(filepath.Join(tmpGit, name))
		}

		// Local config in tmpGit (current working directory) using alternative name
		os.WriteFile(filepath.Join(tmpGit, "git-wt.config.yaml"), []byte(`
hooks:
  add: ["local-add"]
ignore: [".local-ignore"]
ports:
  api: 8080
env:
  LOCAL_ENV: "local-val"
git_hooks:
  enabled: false
  path: .custom-hooks
tmux:
  enabled: true
  keep_open: false
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

		// Ports should be merged
		if AppConfig.Ports["api"] != 8080 {
			t.Errorf("Expected port 8080 for api, got %v", AppConfig.Ports["api"])
		}

		// Env should be loaded (Viper lowercases by default)
		if AppConfig.Env["LOCAL_ENV"] != "local-val" {
			t.Errorf("Expected env LOCAL_ENV 'local-val', got %v", AppConfig.Env["LOCAL_ENV"])
		}
		if AppConfig.GitHooks.Enabled {
			t.Errorf("Expected local git hooks enabled override to be false")
		}
		if AppConfig.GitHooks.Path != ".custom-hooks" {
			t.Errorf("Expected local git hooks path '.custom-hooks', got %q", AppConfig.GitHooks.Path)
		}

		// Tmux keep_open should be merged
		if AppConfig.Tmux.KeepOpen {
			t.Errorf("Expected Tmux.KeepOpen to be false, got %v", AppConfig.Tmux.KeepOpen)
		}
	})

	t.Run("Local legacy githooks alias overrides global git_hooks", func(t *testing.T) {
		configPath, _ := GetGlobalConfigPath()
		configDir := filepath.Dir(configPath)
		os.MkdirAll(configDir, 0755)
		os.WriteFile(configPath, []byte(`
git_hooks:
  enabled: true
  path: .global-hooks
`), 0644)

		for _, name := range LocalConfigNames {
			os.Remove(filepath.Join(tmpGit, name))
		}

		os.WriteFile(filepath.Join(tmpGit, "zgt.config.yaml"), []byte(`
githooks:
  enabled: false
  path: .local-legacy-hooks
`), 0644)

		AppConfig = Config{}
		InitConfig()

		if AppConfig.GitHooks.Enabled {
			t.Errorf("Expected local legacy githooks alias to disable git hooks")
		}
		if AppConfig.GitHooks.Path != ".local-legacy-hooks" {
			t.Errorf("Expected local legacy githooks path '.local-legacy-hooks', got %q", AppConfig.GitHooks.Path)
		}
	})

	t.Run("Explicit config override", func(t *testing.T) {
		explicitPath := filepath.Join(tmpGit, "explicit.yaml")
		os.WriteFile(explicitPath, []byte(`
ports:
  web: 3000
git_hooks:
  enabled: true
`), 0644)

		AppConfig = Config{} // Reset
		CfgFile = explicitPath
		defer func() { CfgFile = "" }()

		InitConfig()

		if AppConfig.Ports["web"] != 3000 {
			t.Errorf("Expected explicit port 3000, got %v", AppConfig.Ports["web"])
		}
		if !AppConfig.GitHooks.Enabled {
			t.Errorf("Expected explicit git hooks to be enabled")
		}
		if AppConfig.GitHooks.Path != ".githooks" {
			t.Errorf("Expected explicit config to inherit default git hooks path, got %q", AppConfig.GitHooks.Path)
		}
	})

	t.Run("Duplicate env keys prevention", func(t *testing.T) {
		tmpRoot, _ := os.MkdirTemp("", "zgt-dup-test")
		defer os.RemoveAll(tmpRoot)

		configPath := filepath.Join(tmpRoot, "zgt.config.yaml")
		_ = os.WriteFile(configPath, []byte(`
env:
  COMPOSE_PROJECT_NAME: "zgt-pj-name"
`), 0644)

		AppConfig = Config{} // Reset
		CfgFile = configPath
		defer func() { CfgFile = "" }()

		InitConfig()

		if _, ok := AppConfig.Env["COMPOSE_PROJECT_NAME"]; !ok {
			t.Errorf("Expected key COMPOSE_PROJECT_NAME to be present")
		}
		if _, ok := AppConfig.Env["compose_project_name"]; ok {
			t.Errorf("Did not expect lowercased duplicate 'compose_project_name' to be present. Keys: %v", getKeys(AppConfig.Env))
		}
	})

	t.Run("GitHooks Shared setting loading", func(t *testing.T) {
		// Remove any existing local config first
		for _, name := range LocalConfigNames {
			os.Remove(filepath.Join(tmpGit, name))
		}

		os.WriteFile(filepath.Join(tmpGit, "zgt.config.yaml"), []byte(`
git_hooks:
  enabled: true
  shared: false
`), 0644)

		AppConfig = Config{}
		InitConfig()

		if !AppConfig.GitHooks.Enabled {
			t.Errorf("Expected git hooks to be enabled")
		}
		if AppConfig.GitHooks.Shared {
			t.Errorf("Expected git hooks shared to be false, got true")
		}
	})

	t.Run("YAML syntax error detection - specific patterns", func(t *testing.T) {
		patterns := []struct {
			name    string
			content string
		}{
			{"Indentation error", "ports:\n  app: 3000\n    sub: 4000"},
			{"Unterminated string", "test: \"aaaaa"},
			{"Mapping without space", "app:3000"}, // This is valid scalar but might fail if structure is expected? Let's check.
		}

		for _, p := range patterns {
			t.Run(p.name, func(t *testing.T) {
				explicitPath := filepath.Join(tmpGit, "invalid_"+p.name+".yaml")
				os.WriteFile(explicitPath, []byte(p.content), 0644)

				AppConfig = Config{} // Reset
				ConfigError = nil    // Reset
				CfgFile = explicitPath
				defer func() { CfgFile = "" }()

				InitConfig()

				// If it's a syntax error, ReadInConfig should set ConfigError.
				// If it's a structural error, Unmarshal might set ConfigError.
				if ConfigError == nil {
					t.Errorf("Expected InitConfig to set ConfigError for %s", p.name)
				}
			})
		}
	})

	t.Run("Empty config file is valid", func(t *testing.T) {
		explicitPath := filepath.Join(tmpGit, "empty.yaml")
		os.WriteFile(explicitPath, []byte(""), 0644)

		AppConfig = Config{} // Reset
		ConfigError = nil    // Reset
		CfgFile = explicitPath
		defer func() { CfgFile = "" }()

		InitConfig()

		if ConfigError != nil {
			t.Errorf("Expected InitConfig to NOT set ConfigError for empty YAML, got: %v", ConfigError)
		}
	})

	t.Run("Local config without tmux.keep_open (defaults to false)", func(t *testing.T) {
		// Remove any existing local config first
		for _, name := range LocalConfigNames {
			os.Remove(filepath.Join(tmpGit, name))
		}

		// Write a local config that has some settings but NOT tmux.auto_close
		os.WriteFile(filepath.Join(tmpGit, "zgt.config.yml"), []byte(`
hooks:
  add: ["local-only-add"]
`), 0644)

		// Also ensure global config exists and has auto_close = true
		configPath, _ := GetGlobalConfigPath()
		configDir := filepath.Dir(configPath)
		os.MkdirAll(configDir, 0755)
		os.WriteFile(configPath, []byte(`
tmux:
  keep_open: false
`), 0644)

		AppConfig = Config{} // Reset
		InitConfig()

		if AppConfig.Tmux.KeepOpen {
			t.Errorf("Expected Tmux.KeepOpen to remain false when not specified in local config, but got true")
		}
	})

	t.Run("Local config with explicit tmux.keep_open: true", func(t *testing.T) {
		// Remove any existing local config first
		for _, name := range LocalConfigNames {
			os.Remove(filepath.Join(tmpGit, name))
		}

		// Write a local config that has tmux.auto_close: false
		os.WriteFile(filepath.Join(tmpGit, "zgt.config.yml"), []byte(`
tmux:
  keep_open: true
`), 0644)

		AppConfig = Config{} // Reset
		InitConfig()

		if !AppConfig.Tmux.KeepOpen {
			t.Errorf("Expected Tmux.KeepOpen to be true when explicitly set in local config, but got false")
		}
	})
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
