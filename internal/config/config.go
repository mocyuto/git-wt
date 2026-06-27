package config

import (
	"io"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mocyuto/zgt/internal/gitroot"
	"github.com/mocyuto/zgt/internal/logger"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	GlobalConfigDir  = ".config/zgt"
	GlobalConfigName = "config"
	GlobalConfigExt  = "yaml"

	DefaultLocalConfigName = "zgt.config.yml"
)

var (
	LocalConfigNames = []string{
		"zgt.config.yml",
		"zgt.config.yaml",
		"git-wt.config.yml",
		"git-wt.config.yaml",
	}
)

func GetGlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, GlobalConfigDir, GlobalConfigName+"."+GlobalConfigExt), nil
}

func GetLocalConfigPath(root string) string {
	for _, name := range LocalConfigNames {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

var (
	CfgFile     string
	ConfigError error
)

type TmuxPane struct {
	Id       string   `mapstructure:"id"`
	Target   string   `mapstructure:"target"`
	Commands []string `mapstructure:"commands"`
	Split    string   `mapstructure:"split"`
	Size     string   `mapstructure:"size"`
}

type TmuxConfig struct {
	Enabled    bool       `mapstructure:"enabled"`
	KeepOpen   bool       `mapstructure:"keep_open"`
	WindowName string     `mapstructure:"window_name"`
	Panes      []TmuxPane `mapstructure:"panes"`
}

type GitHooksConfig struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
	Path    string `mapstructure:"path" yaml:"path"`
	Shared  bool   `mapstructure:"shared" yaml:"shared"`
}

// HooksConfig holds lifecycle hook commands for the add/remove commands.
type HooksConfig struct {
	Add []string `mapstructure:"add" yaml:"add"`
	RM  []string `mapstructure:"rm"  yaml:"rm"`
}

// ProfileConfig is a profile-specific override applied on top of the
// top-level Config. Profile env keys override top-level env per-key; profile
// hooks.add / hooks.rm are appended AFTER the top-level hooks. Profile tmux
// is merged per-field onto the top-level Tmux (see ProfileTmux).
type ProfileConfig struct {
	Env   map[string]string `mapstructure:"env"   yaml:"env"`
	Hooks HooksConfig       `mapstructure:"hooks" yaml:"hooks"`
	Tmux  TmuxConfig        `mapstructure:"tmux"  yaml:"tmux"`
}

type Config struct {
	Hooks HooksConfig `mapstructure:"hooks" yaml:"hooks"`
	Add   struct {
		FromDefault bool `mapstructure:"from_default"`
		AutoPull    bool `mapstructure:"auto_pull"`
	} `mapstructure:"add"`
	Ignore   []string                 `mapstructure:"ignore"`
	Ports    map[string]int           `mapstructure:"ports"`
	Env      map[string]string        `mapstructure:"env"`
	Tmux     TmuxConfig               `mapstructure:"tmux"`
	GitHooks GitHooksConfig           `mapstructure:"git_hooks" yaml:"git_hooks"`
	Profiles map[string]ProfileConfig `mapstructure:"profiles" yaml:"profiles"`
}

var AppConfig Config

func InitConfig() {
	// Initialize default viper for global/local loading
	v := viper.New()
	setViperDefaults(v)

	// 1. Load global config
	globalPath, err := GetGlobalConfigPath()
	if err == nil {
		configDir := filepath.Dir(globalPath)
		v.AddConfigPath(configDir)
		v.SetConfigName(GlobalConfigName)
		v.SetConfigType(GlobalConfigExt)

		// Create default global config if it doesn't exist
		if _, err := os.Stat(globalPath); os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err == nil {
				_ = v.SafeWriteConfigAs(globalPath)
			}
		}

		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				ConfigError = logger.Errorf("reading global config: %v", err)
			}
		}
	}

	// Unmarshal global config into AppConfig
	if err := v.Unmarshal(&AppConfig); err != nil && ConfigError == nil {
		ConfigError = logger.Errorf("unmarshaling global config: %v", err)
	}
	applyDefaults(&AppConfig)

	// Fix case for global env if loaded
	if globalPath != "" {
		if _, err := os.Stat(globalPath); err == nil {
			if err := loadEnvCasePreserved(globalPath); err != nil {
				logger.Warn("failed to restore environment variable case in global config: %v", err)
			}
		}
	}

	// 2. Load local config
	gitRoot, _ := gitroot.GetMainProjectRoot()
	if gitRoot != "" {
		localPath := GetLocalConfigPath(gitRoot)
		if localPath != "" {
			localV := viper.New()
			localV.SetConfigFile(localPath)

			if err := localV.ReadInConfig(); err != nil {
				ConfigError = logger.Errorf("reading local config: %v", err)
			}

			// Get raw map to check for presence of keys (booleans default to false on unmarshal)
			raw := localV.AllSettings()

			var localConfig Config
			if err := localV.Unmarshal(&localConfig); err == nil {
				// Merge hooks
				AppConfig.Hooks.Add = append(AppConfig.Hooks.Add, localConfig.Hooks.Add...)
				AppConfig.Hooks.RM = append(AppConfig.Hooks.RM, localConfig.Hooks.RM...)
				// Merge ignores
				AppConfig.Ignore = append(AppConfig.Ignore, localConfig.Ignore...)
				// Merge add config
				if localConfig.Add.FromDefault {
					AppConfig.Add.FromDefault = true
				}
				if localConfig.Add.AutoPull {
					AppConfig.Add.AutoPull = true
				}
				// Merge ports
				if AppConfig.Ports == nil {
					AppConfig.Ports = make(map[string]int)
				}
				maps.Copy(AppConfig.Ports, localConfig.Ports)
				// Merge env
				if AppConfig.Env == nil {
					AppConfig.Env = make(map[string]string)
				}
				maps.Copy(AppConfig.Env, localConfig.Env)

				// Merge profiles:
				//   * A profile the global config does NOT define is added verbatim.
				//   * A profile the global config DOES define gets its env merged per-key
				//     (local overrides global) and its hooks appended AFTER the global
				//     profile's hooks (mirroring the top-level vs profile ordering: global
				//     top-level hooks run first, then global profile hooks, then local profile
				//     hooks).
				if len(localConfig.Profiles) > 0 {
					if AppConfig.Profiles == nil {
						AppConfig.Profiles = make(map[string]ProfileConfig)
					}
					for name, localP := range localConfig.Profiles {
						globalP, ok := AppConfig.Profiles[name]
						if !ok {
							AppConfig.Profiles[name] = localP
							continue
						}
						if len(localP.Env) > 0 {
							if globalP.Env == nil {
								globalP.Env = make(map[string]string)
							}
							for k, v := range localP.Env {
								globalP.Env[k] = v
							}
						}
						if len(localP.Hooks.Add) > 0 {
							merged := make([]string, 0, len(globalP.Hooks.Add)+len(localP.Hooks.Add))
							merged = append(merged, globalP.Hooks.Add...)
							merged = append(merged, localP.Hooks.Add...)
							globalP.Hooks.Add = merged
						}
						if len(localP.Hooks.RM) > 0 {
							merged := make([]string, 0, len(globalP.Hooks.RM)+len(localP.Hooks.RM))
							merged = append(merged, globalP.Hooks.RM...)
							merged = append(merged, localP.Hooks.RM...)
							globalP.Hooks.RM = merged
						}
						// Merge profile-level tmux with the same per-field
						// semantics as ProfileTmux (single source of truth:
						// mergeTmux). Enabled/KeepOpen are OR-ed, WindowName is
						// overridden when non-empty, and Panes are replaced
						// wholesale when the local profile defines any.
						globalP.Tmux = mergeTmux(globalP.Tmux, localP.Tmux)
						AppConfig.Profiles[name] = globalP
					}
				}

				// Merge git hooks
				if hasKey(raw, "git_hooks", "enabled") {
					AppConfig.GitHooks.Enabled = localConfig.GitHooks.Enabled
				}
				if localConfig.GitHooks.Path != "" {
					AppConfig.GitHooks.Path = localConfig.GitHooks.Path
				}
				if hasKey(raw, "git_hooks", "shared") {
					AppConfig.GitHooks.Shared = localConfig.GitHooks.Shared
				}

				// Merge tmux
				if localConfig.Tmux.Enabled {
					AppConfig.Tmux.Enabled = true
				}
				// KeepOpen is now default false in global config.
				// We only override it if it's explicitly set in local config.
				if hasKey(raw, "tmux", "keep_open") {
					AppConfig.Tmux.KeepOpen = localConfig.Tmux.KeepOpen
				}
				if len(localConfig.Tmux.Panes) > 0 {
					AppConfig.Tmux.Panes = localConfig.Tmux.Panes
				}

				// Fix case for local env
				if err := loadEnvCasePreserved(localPath); err != nil {
					logger.Warn("failed to restore environment variable case in local config: %v", err)
				}
				applyDefaults(&AppConfig)
			} else if ConfigError == nil {
				ConfigError = logger.Errorf("unmarshaling local config: %v", err)
			}
		}
	}

	// 3. Override with --config if provided
	if CfgFile != "" {
		explicitV := viper.New()
		setViperDefaults(explicitV)
		explicitV.SetConfigFile(CfgFile)
		if err := explicitV.ReadInConfig(); err != nil {
			ConfigError = logger.Errorf("reading explicit config: %v", err)
		} else {
			// Full override if specific config is provided
			var explicitConfig Config
			if err := explicitV.Unmarshal(&explicitConfig); err == nil {
				AppConfig = explicitConfig
				// Fix case for explicit env
				if err := loadEnvCasePreserved(CfgFile); err != nil {
					logger.Warn("failed to restore environment variable case in explicit config: %v", err)
				}
				applyDefaults(&AppConfig)
			} else if ConfigError == nil {
				ConfigError = logger.Errorf("unmarshaling explicit config: %v", err)
			}
		}
	}
}

func setViperDefaults(v *viper.Viper) {
	v.SetDefault("hooks.add", []string{})
	v.SetDefault("hooks.rm", []string{})
	v.SetDefault("ignore", []string{})
	v.SetDefault("git_hooks.enabled", false)
	v.SetDefault("git_hooks.path", ".githooks")
	v.SetDefault("git_hooks.shared", true)
	v.SetDefault("tmux.keep_open", false)
}

func applyDefaults(cfg *Config) {
	if cfg.GitHooks.Path == "" {
		cfg.GitHooks.Path = ".githooks"
	}
	// Note: We don't force Shared=true here because it might have been set to false by unmarshal
}

// loadEnvCasePreserved re-reads the config file using yaml.v3 to preserve case in env map
// (top-level env and per-profile env).
func loadEnvCasePreserved(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var raw map[string]interface{}
	if err := yaml.NewDecoder(f).Decode(&raw); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	// Top-level env
	if envRaw, ok := raw["env"]; ok {
		applyEnvCase(envRaw, &AppConfig.Env)
	}

	// Per-profile env
	if profilesRaw, ok := raw["profiles"]; ok {
		if profilesMap, ok := profilesRaw.(map[string]interface{}); ok {
			if AppConfig.Profiles == nil {
				AppConfig.Profiles = make(map[string]ProfileConfig)
			}
			for name, pRaw := range profilesMap {
				pMap, ok := pRaw.(map[string]interface{})
				if !ok {
					continue
				}
				// Viper has already populated AppConfig.Profiles under a
				// lowercased key (and lowercased env keys). Update the
				// existing entry in place rather than creating a new entry
				// under the original-case name — otherwise mixed-case
				// profile names silently split into two entries and the
				// viper-derived hooks end up dangling. We MERGE env keys
				// per-key (preserve globally-defined keys, fix case on local
				// keys) so global+local configs that define the same
				// profile env keys compose correctly.
				lowerName := strings.ToLower(name)
				p, ok := AppConfig.Profiles[lowerName]
				if !ok {
					p = ProfileConfig{}
				}
				if p.Env == nil {
					p.Env = make(map[string]string)
				}
				if envRaw, ok := pMap["env"]; ok {
					applyEnvCase(envRaw, &p.Env)
				}
				AppConfig.Profiles[lowerName] = p
			}
		}
	}
	return nil
}

// applyEnvCase merges env entries from raw into dst, preserving the case of
// keys and removing any lowercased duplicates that Viper may have introduced.
// Non-string YAML scalars (int / bool / float) are stringified via fmt.Sprint,
// matching Viper's mapstructure coercion into map[string]string.
func applyEnvCase(raw any, dst *map[string]string) {
	envMap, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	if *dst == nil {
		*dst = make(map[string]string)
	}
	for k, v := range envMap {
		val := scalarToString(v)
		if val == "" && v != nil && v != "" {
			// Non-scalar value (list/map); skip - env maps only hold scalars.
			continue
		}
		lowerK := strings.ToLower(k)
		if _, exists := (*dst)[lowerK]; exists && lowerK != k {
			delete(*dst, lowerK)
		}
		(*dst)[k] = val
	}
}

// scalarToString converts a YAML scalar to its string form. Returns "" for
// non-scalar values (arrays, maps). nil -> "".
func scalarToString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		// Format without trailing zeros for typical integer-valued floats.
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		// Anything that yaml.v3 decoded as a list or map is non-scalar.
		return ""
	}
}

// ProfileHooks returns the merged hook commands for a given action ("add" or
// "rm") and profile. Top-level hooks are always included; profile-specific
// hooks are appended afterwards. An empty or "default" profile yields the
// top-level hooks only. Unknown profiles fall back to top-level behavior.
// Profile name lookup is case-insensitive (profile names are stored in
// lower-case to match Viper's map-key normalization).
func ProfileHooks(action, profile string) []string {
	var base []string
	switch action {
	case "add":
		base = AppConfig.Hooks.Add
	case "rm":
		base = AppConfig.Hooks.RM
	}
	p, ok := lookupProfile(profile)
	if !ok {
		return base
	}
	var extra []string
	switch action {
	case "add":
		extra = p.Hooks.Add
	case "rm":
		extra = p.Hooks.RM
	}
	if len(extra) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

// ProfileEnv returns the merged env map for a profile. Top-level env is used
// as base; profile env keys override per-key. The returned map is always a
// fresh copy and safe to mutate.
func ProfileEnv(profile string) map[string]string {
	base := make(map[string]string, len(AppConfig.Env))
	for k, v := range AppConfig.Env {
		base[k] = v
	}
	p, ok := lookupProfile(profile)
	if !ok {
		return base
	}
	for k, v := range p.Env {
		base[k] = v
	}
	return base
}

// ProfileTmux returns the merged TmuxConfig for a profile. The top-level
// Tmux is used as base; the profile's Tmux is overlaid per-field:
//   - Enabled and KeepOpen use OR semantics (profile can enable but not
//     disable).
//   - WindowName is overridden when the profile sets a non-empty value.
//   - Panes are replaced wholesale when the profile defines at least one
//     pane.
//
// An empty or "default" profile yields the top-level Tmux unchanged. The
// returned Panes slice is always a fresh copy, so callers may mutate it
// without affecting AppConfig.
func ProfileTmux(profile string) TmuxConfig {
	p, ok := lookupProfile(profile)
	if !ok {
		return mergeTmux(AppConfig.Tmux, TmuxConfig{})
	}
	return mergeTmux(AppConfig.Tmux, p.Tmux)
}

// mergeTmux overlays src onto dst per-field with the same rules as
// ProfileTmux: Enabled/KeepOpen are OR-ed, WindowName is overridden when
// non-empty, and Panes are replaced when src defines any. The returned
// Panes slice is always a fresh copy so the result is safe for callers to
// mutate without aliasing AppConfig.
func mergeTmux(dst, src TmuxConfig) TmuxConfig {
	if src.Enabled {
		dst.Enabled = true
	}
	if src.KeepOpen {
		dst.KeepOpen = true
	}
	if src.WindowName != "" {
		dst.WindowName = src.WindowName
	}
	switch {
	case len(src.Panes) > 0:
		dst.Panes = append([]TmuxPane(nil), src.Panes...)
	case len(dst.Panes) > 0:
		dst.Panes = append([]TmuxPane(nil), dst.Panes...)
	default:
		dst.Panes = nil
	}
	return dst
}

// ProfileExists returns true if the named profile is defined in the config.
// Empty and "default" are always considered valid. Profile name lookup is
// case-insensitive.
func ProfileExists(profile string) bool {
	if profile == "" || profile == "default" {
		return true
	}
	_, ok := lookupProfile(profile)
	return ok
}

// ProfileNames returns the names of all profiles defined in the config, sorted
// alphabetically, with the implicit "default" entry included. Profile names
// are stored lowercased internally (Viper normalizes map keys), so the
// returned names are lowercased. The list is safe to use as shell-completion
// candidates for the --profile flag and contains only "default" when no
// profiles section is defined. A user-defined `profiles.default` entry is
// suppressed here (it is unreachable at runtime because lookupProfile treats
// "default" as the implicit no-op profile), so "default" always appears at
// most once.
func ProfileNames() []string {
	names := make([]string, 0, len(AppConfig.Profiles)+1)
	for name := range AppConfig.Profiles {
		if name == "default" {
			continue
		}
		names = append(names, name)
	}
	names = append(names, "default")
	sort.Strings(names)
	return names
}

// lookupProfile returns the ProfileConfig for the given name (case-insensitive
// match against the lower-cased profile keys), or (zero, false) when the name
// is empty, "default", or not defined.
func lookupProfile(profile string) (ProfileConfig, bool) {
	if profile == "" || profile == "default" || AppConfig.Profiles == nil {
		return ProfileConfig{}, false
	}
	p, ok := AppConfig.Profiles[strings.ToLower(profile)]
	return p, ok
}

// hasKey checks if a nested key exists in a map[string]interface{}
func hasKey(m map[string]any, keys ...string) bool {
	curr := m
	for i, k := range keys {
		val, ok := curr[k]
		if !ok {
			return false
		}
		if i == len(keys)-1 {
			return true
		}
		next, ok := val.(map[string]any)
		if !ok {
			return false
		}
		curr = next
	}
	return false
}

// LoadPortsFromPath loads only the ports configuration from a zgt.config.yml/yaml in the given directory
func LoadPortsFromPath(root string) (map[string]int, error) {
	localPath := GetLocalConfigPath(root)
	if localPath == "" {
		return nil, logger.Errorf("no config file found in path %s", root)
	}

	v := viper.New()
	v.SetConfigFile(localPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, logger.Errorf("reading config from path %s: %v", localPath, err)
	}

	var cfg struct {
		Ports map[string]int `mapstructure:"ports"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, logger.Errorf("unmarshaling config from path %s: %v", localPath, err)
	}

	return cfg.Ports, nil
}

// PrintConfig prints the current configuration to the log
func PrintConfig() {
	logger.Info("--- Current Configuration ---")
	d, err := yaml.Marshal(&AppConfig)
	if err != nil {
		logger.Warn("failed to marshal config: %v", err)
		return
	}
	logger.Log(string(d))
	logger.Info("-----------------------------")
}
