package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	t.Run("GitHooks Shared defaults to true when omitted", func(t *testing.T) {
		// Set up global config that omits 'shared'
		configPath, _ := GetGlobalConfigPath()
		configDir := filepath.Dir(configPath)
		os.MkdirAll(configDir, 0755)
		os.WriteFile(configPath, []byte(`
git_hooks:
  enabled: true
`), 0644)

		for _, name := range LocalConfigNames {
			os.Remove(filepath.Join(tmpGit, name))
		}

		AppConfig = Config{}
		InitConfig()

		if !AppConfig.GitHooks.Enabled {
			t.Errorf("Expected git hooks to be enabled")
		}
		if !AppConfig.GitHooks.Shared {
			t.Errorf("Expected git hooks shared to default to true, got false")
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

func TestInitConfig_Profiles_MixedCase(t *testing.T) {
	tmpGit, err := os.MkdirTemp("", "zgt-profile-case-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpGit)

	exec.Command("git", "init", tmpGit).Run()

	configPath := filepath.Join(tmpGit, "zgt.config.yaml")
	_ = os.WriteFile(configPath, []byte(`
env:
  TopKey: "base"
profiles:
  MixedCase:
    env:
      MyVar: "y"
    hooks:
      add:
        - "echo hi"
`), 0644)

	AppConfig = Config{}
	ConfigError = nil
	CfgFile = configPath
	defer func() { CfgFile = "" }()

	InitConfig()

	if len(AppConfig.Profiles) != 1 {
		t.Fatalf("expected exactly 1 profile entry (lowercased), got %d: %v", len(AppConfig.Profiles), AppConfig.Profiles)
	}
	// Only the lowercased key should remain; no split MixedCase + mixedcase entry.
	if _, exists := AppConfig.Profiles["mixedcase"]; !exists {
		t.Errorf("expected lowercased 'mixedcase' key, got: %v", AppConfig.Profiles)
	}
	if _, exists := AppConfig.Profiles["MixedCase"]; exists {
		t.Errorf("expected original-case 'MixedCase' entry to NOT exist, got: %v", AppConfig.Profiles)
	}

	// Hooks should survive the case fix (regression for H1).
	got := ProfileHooks("add", "MixedCase")
	if len(got) != 1 || got[0] != "echo hi" {
		t.Errorf("ProfileHooks(add, MixedCase) = %v, want [echo hi]", got)
	}
	gotLower := ProfileHooks("add", "mixedcase")
	if len(gotLower) != 1 || gotLower[0] != "echo hi" {
		t.Errorf("ProfileHooks(add, mixedcase) = %v, want [echo hi]", gotLower)
	}

	// Env case should be preserved; lookups are case-insensitive on profile name.
	env := ProfileEnv("MixedCase")
	if env["MyVar"] != "y" {
		t.Errorf("ProfileEnv(MixedCase)[MyVar] = %q, want %q (case preserved)", env["MyVar"], "y")
	}
	if _, exists := env["myvar"]; exists {
		t.Errorf("ProfileEnv(MixedCase) should NOT contain lowercased 'myvar': %v", env)
	}
	if env["TopKey"] != "base" {
		t.Errorf("ProfileEnv(MixedCase)[TopKey] = %q, want %q (top-level inherited)", env["TopKey"], "base")
	}
	if !ProfileExists("MixedCase") || !ProfileExists("mixedcase") {
		t.Errorf("ProfileExists should be case-insensitive; MixedCase/mixedcase both expected to resolve")
	}
	if ProfileExists("OTHER") {
		t.Errorf("ProfileExists(OTHER) should be false")
	}
}

func TestInitConfig_Profiles_FromLocalConfig(t *testing.T) {
	// Regression for the "local zgt.config.yaml profiles silently dropped"
	// bug: local config merge never copied AppConfig.Profiles from the
	// freshly unmarshaled localConfig.
	tmpGit, err := os.MkdirTemp("", "zgt-local-profile-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpGit)
	exec.Command("git", "init", tmpGit).Run()

	// Global config without profiles
	tmpHome, err := os.MkdirTemp("", "zgt-local-profile-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	origWd, _ := os.Getwd()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// HOME = tmpHome -> global dir = tmpHome/.config/zgt
	globalDir := filepath.Join(tmpHome, ".config", "zgt")
	os.MkdirAll(globalDir, 0755)
	os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(`hooks: { add: ["global"] }`), 0644)

	// Local config with a migration profile
	localPath := filepath.Join(tmpGit, "zgt.config.yaml")
	os.WriteFile(localPath, []byte(`
profiles:
  migration:
    env:
      DB_HOST: "iso-db"
    hooks:
      add:
        - "./scripts/db-clone.sh"
      rm:
        - "docker compose down -v"
`), 0644)

	os.Chdir(tmpGit)
	defer os.Chdir(origWd)

	AppConfig = Config{}
	ConfigError = nil
	InitConfig()

	if ConfigError != nil {
		t.Fatalf("InitConfig failed: %v", ConfigError)
	}
	p, ok := AppConfig.Profiles["migration"]
	if !ok {
		t.Fatalf("migration profile was not merged from local config; profiles=%v", AppConfig.Profiles)
	}
	// Env case must be preserved by loadEnvCasePreserved.
	if p.Env["DB_HOST"] != "iso-db" {
		t.Errorf("profile env DB_HOST = %q, want %q", p.Env["DB_HOST"], "iso-db")
	}
	if p.Hooks.Add[0] != "./scripts/db-clone.sh" {
		t.Errorf("profile hooks.add = %v, want [./scripts/db-clone.sh]", p.Hooks.Add)
	}
	if p.Hooks.RM[0] != "docker compose down -v" {
		t.Errorf("profile hooks.rm = %v, want [docker compose down -v]", p.Hooks.RM)
	}

	// Hook resolution should see the profile hooks.
	if got := ProfileHooks("add", "migration"); len(got) != 2 || got[0] != "global" || got[1] != "./scripts/db-clone.sh" {
		t.Errorf("ProfileHooks(add, migration) = %v, want [global ./scripts/db-clone.sh]", got)
	}
}

// TestInitConfig_Profiles_GlobalPlusLocalMerge verifies that a profile
// defined in BOTH the global config and the local config composes rather
// than the local one silently replacing the global one (regression for the
// "global profile hooks silently lost" issue). Hooks append; env merges
// per-key with local overriding global.
func TestInitConfig_Profiles_GlobalPlusLocalMerge(t *testing.T) {
	tmpGit, err := os.MkdirTemp("", "zgt-gp-local-merge")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpGit)
	exec.Command("git", "init", tmpGit).Run()

	tmpHome, err := os.MkdirTemp("", "zgt-gp-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	origWd, _ := os.Getwd()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	globalDir := filepath.Join(tmpHome, ".config", "zgt")
	os.MkdirAll(globalDir, 0755)
	// Global migration profile: initializes the DB.
	os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(`
profiles:
  migration:
    env:
      MAX_CONNS: "50"
      SHARED_KEY: "global"
    hooks:
      add:
        - "global-mig-init"
`), 0644)

	localPath := filepath.Join(tmpGit, "zgt.config.yaml")
	// Local migration profile: adds a hook and overrides SHARED_KEY while
	// leaving MAX_CONNS untouched, and adds an extra env var.
	os.WriteFile(localPath, []byte(`
profiles:
  migration:
    env:
      SHARED_KEY: "local"
      EXTRA: "local-val"
    hooks:
      add:
        - "local-mig-extension"
`), 0644)

	os.Chdir(tmpGit)
	defer os.Chdir(origWd)

	AppConfig = Config{}
	ConfigError = nil
	InitConfig()

	if ConfigError != nil {
		t.Fatalf("InitConfig failed: %v", ConfigError)
	}
	p, ok := AppConfig.Profiles["migration"]
	if !ok {
		t.Fatalf("migration profile not found: %v", AppConfig.Profiles)
	}

	// Env: MAX_CONNS inherited from global, SHARED_KEY overridden by local,
	// EXTRA added by local.
	if p.Env["MAX_CONNS"] != "50" {
		t.Errorf("MAX_CONNS dropped; env = %v", p.Env)
	}
	if p.Env["SHARED_KEY"] != "local" {
		t.Errorf("SHARED_KEY not overridden; env = %v", p.Env)
	}
	if p.Env["EXTRA"] != "local-val" {
		t.Errorf("EXTRA not added; env = %v", p.Env)
	}

	// Hooks: global profile hook runs FIRST, then local (matching the
	// top-level -> profile ordering).
	if got := ProfileHooks("add", "migration"); len(got) != 2 || got[0] != "global-mig-init" || got[1] != "local-mig-extension" {
		t.Errorf("ProfileHooks(add, migration) = %v, want [global-mig-init local-mig-extension]", got)
	}
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestProfileHelpers(t *testing.T) {
	AppConfig = Config{}
	AppConfig.Env = map[string]string{
		"COMPOSE_PROJECT_NAME": "{{.Repo}}-{{.Branch}}",
		"DB_HOST":              "db",
	}
	AppConfig.Hooks.Add = []string{"docker compose up -d api envoy"}
	AppConfig.Hooks.RM = []string{"docker compose down"}
	AppConfig.Profiles = map[string]ProfileConfig{
		"migration": {
			Env: map[string]string{
				"DB_HOST": "{{.Repo}}-migration-db",
			},
			Hooks: HooksConfig{
				Add: []string{"./scripts/db-clone.sh"},
				RM:  []string{"docker compose down -v"},
			},
		},
	}

	// ProfileEnv: default returns top-level only
	defEnv := ProfileEnv("")
	if defEnv["DB_HOST"] != "db" {
		t.Errorf("ProfileEnv(\"\") DB_HOST = %q, want %q", defEnv["DB_HOST"], "db")
	}
	if defEnv["COMPOSE_PROJECT_NAME"] != "{{.Repo}}-{{.Branch}}" {
		t.Errorf("ProfileEnv(\"\") COMPOSE_PROJECT_NAME missing: %v", defEnv)
	}

	// ProfileEnv: migration overrides DB_HOST, keeps COMPOSE_PROJECT_NAME
	mEnv := ProfileEnv("migration")
	if mEnv["DB_HOST"] != "{{.Repo}}-migration-db" {
		t.Errorf("ProfileEnv(migration) DB_HOST = %q, want overridden", mEnv["DB_HOST"])
	}
	if mEnv["COMPOSE_PROJECT_NAME"] != "{{.Repo}}-{{.Branch}}" {
		t.Errorf("ProfileEnv(migration) lost COMPOSE_PROJECT_NAME: %v", mEnv)
	}

	// ProfileEnv: unknown profile falls back to top-level
	unkEnv := ProfileEnv("nonexistent")
	if unkEnv["DB_HOST"] != "db" {
		t.Errorf("ProfileEnv(unknown) DB_HOST = %q, want fallback %q", unkEnv["DB_HOST"], "db")
	}

	// Returned map is a copy (mutation does not affect AppConfig)
	unkEnv["NEW_KEY"] = "x"
	if _, exists := AppConfig.Env["NEW_KEY"]; exists {
		t.Errorf("ProfileEnv returned a shared map; mutation leaked to AppConfig")
	}

	// ProfileHooks: default returns top-level only
	if got := ProfileHooks("add", ""); len(got) != 1 || got[0] != "docker compose up -d api envoy" {
		t.Errorf("ProfileHooks(add, default) = %v", got)
	}
	// ProfileHooks: migration appends profile hook after top-level
	got := ProfileHooks("add", "migration")
	if len(got) != 2 {
		t.Fatalf("ProfileHooks(add, migration) len = %d, want 2: %v", len(got), got)
	}
	if got[0] != "docker compose up -d api envoy" {
		t.Errorf("first migration add hook should be top-level: %v", got)
	}
	if got[1] != "./scripts/db-clone.sh" {
		t.Errorf("second migration add hook should be profile-specific: %v", got)
	}
	// ProfileHooks: rm appends
	gotRM := ProfileHooks("rm", "migration")
	if len(gotRM) != 2 || gotRM[0] != "docker compose down" || gotRM[1] != "docker compose down -v" {
		t.Errorf("ProfileHooks(rm, migration) = %v", gotRM)
	}
	// ProfileHooks: unknown profile falls back
	if got := ProfileHooks("add", "nonexistent"); len(got) != 1 {
		t.Errorf("ProfileHooks(add, unknown) should fall back: %v", got)
	}

	// ProfileExists
	if !ProfileExists("") {
		t.Errorf("ProfileExists(\"\") should be true")
	}
	if !ProfileExists("default") {
		t.Errorf("ProfileExists(default) should be true")
	}
	if !ProfileExists("migration") {
		t.Errorf("ProfileExists(migration) should be true")
	}
	if ProfileExists("nonexistent") {
		t.Errorf("ProfileExists(nonexistent) should be false")
	}
}

func TestInitConfig_Profiles(t *testing.T) {
	// Set up a temp project with a profile config
	tmpGit, err := os.MkdirTemp("", "zgt-profile-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpGit)

	exec.Command("git", "init", tmpGit).Run()

	configPath := filepath.Join(tmpGit, "zgt.config.yaml")
	_ = os.WriteFile(configPath, []byte(`
env:
  COMPOSE_PROJECT_NAME: "{{.Repo}}-{{.Branch}}"
profiles:
  migration:
    env:
      DB_HOST: "iso-db"
      COMPOSE_PROJECT_NAME: "{{.Repo}}-migration-{{.Branch}}"
    hooks:
      add:
        - "./scripts/db-clone.sh"
      rm:
        - "docker compose down -v"
`), 0644)

	AppConfig = Config{}
	ConfigError = nil
	CfgFile = configPath
	defer func() { CfgFile = "" }()

	InitConfig()

	if ConfigError != nil {
		t.Fatalf("InitConfig failed: %v", ConfigError)
	}
	if AppConfig.Profiles == nil {
		t.Fatalf("Profiles not loaded")
	}
	p, ok := AppConfig.Profiles["migration"]
	if !ok {
		t.Fatalf("migration profile not found; profiles=%v", AppConfig.Profiles)
	}
	// Env case preserved
	if p.Env["DB_HOST"] != "iso-db" {
		t.Errorf("profile env DB_HOST = %q, want %q", p.Env["DB_HOST"], "iso-db")
	}
	if p.Env["COMPOSE_PROJECT_NAME"] != "{{.Repo}}-migration-{{.Branch}}" {
		t.Errorf("profile env COMPOSE_PROJECT_NAME = %q", p.Env["COMPOSE_PROJECT_NAME"])
	}
	// No lower-cased dup
	if _, exists := p.Env["compose_project_name"]; exists {
		t.Errorf("lowercased duplicate compose_project_name should not exist: %v", p.Env)
	}
	// Hooks loaded
	if len(p.Hooks.Add) != 1 || p.Hooks.Add[0] != "./scripts/db-clone.sh" {
		t.Errorf("profile hooks.add = %v", p.Hooks.Add)
	}
	if len(p.Hooks.RM) != 1 || p.Hooks.RM[0] != "docker compose down -v" {
		t.Errorf("profile hooks.rm = %v", p.Hooks.RM)
	}
}

// TestInitConfig_Profiles_NonStringEnv verifies that non-string YAML scalars
// in profile env (int/bool/float) survive Viper round-tripping via the
// mapstructure string coercion; applyEnvCase must not DROP them when fixing
// case (regression for ROUND-2 HIGH-2).
func TestInitConfig_Profiles_NonStringEnv(t *testing.T) {
	tmpGit, err := os.MkdirTemp("", "zgt-profile-nonstring-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpGit)
	exec.Command("git", "init", tmpGit).Run()

	configPath := filepath.Join(tmpGit, "zgt.config.yaml")
	// YAML 1.2 bool/int/coercion: true -> "true", 5 -> "5", 1.5 -> "1.5"
	os.WriteFile(configPath, []byte(`
profiles:
  migration:
    env:
      MAX_CONNS: 100
      DEBUG: true
      RATIO: 1.5
      PAUSED: false
      NAME: "iso-db"
`), 0644)

	AppConfig = Config{}
	ConfigError = nil
	CfgFile = configPath
	defer func() { CfgFile = "" }()

	InitConfig()

	if ConfigError != nil {
		t.Fatalf("InitConfig failed: %v", ConfigError)
	}
	p, ok := AppConfig.Profiles["migration"]
	if !ok {
		t.Fatalf("migration profile not loaded: %v", AppConfig.Profiles)
	}
	cases := map[string]string{
		"MAX_CONNS": "100",
		"DEBUG":     "true",
		"RATIO":     "1.5",
		"PAUSED":    "false",
		"NAME":      "iso-db",
	}
	for k, want := range cases {
		if got := p.Env[k]; got != want {
			t.Errorf("env[%s] = %q, want %q (case preserved, value coerced)", k, got, want)
		}
		// And no lowercased dup lingered:
		if _, exists := p.Env[strings.ToLower(k)]; exists && strings.ToLower(k) != k {
			t.Errorf("lowercased dup %q still present: %v", strings.ToLower(k), p.Env)
		}
	}
}
