package config

import (
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/mocyuto/git-wt/internal/git"
	"github.com/mocyuto/git-wt/internal/logger"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	CfgFile     string
	ConfigError error
)

type Config struct {
	Hooks struct {
		Add []string `mapstructure:"add"`
		RM  []string `mapstructure:"rm"`
	} `mapstructure:"hooks"`
	Ignore []string          `mapstructure:"ignore"`
	Ports  map[string]int    `mapstructure:"ports"`
	Env    map[string]string `mapstructure:"env"`
}

var AppConfig Config

func InitConfig() {
	// Initialize default viper for global/local loading
	v := viper.New()

	// 1. Load global config
	var globalPath string
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config", "git-wt")
		v.AddConfigPath(configDir)
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Create default global config if it doesn't exist
		globalPath = filepath.Join(configDir, "config.yaml")
		if _, err := os.Stat(globalPath); os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err == nil {
				v.SetDefault("hooks.add", []string{})
				v.SetDefault("hooks.rm", []string{})
				v.SetDefault("ignore", []string{})
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

	// Fix case for global env if loaded
	if globalPath != "" {
		if err := loadEnvCasePreserved(globalPath); err != nil {
			logger.Warn("failed to restore environment variable case in global config: %v", err)
		}
	}

	// 2. Load local config
	gitRoot, _ := git.GetMainProjectRoot()
	if gitRoot != "" {
		localV := viper.New()
		localV.AddConfigPath(gitRoot)
		localV.SetConfigName("git-wt.config")
		localV.SetConfigType("yaml")

		var localPath string
		if err := localV.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				ConfigError = logger.Errorf("reading local config: %v", err)
			}
		} else {
			localPath = localV.ConfigFileUsed()
		}

		var localConfig Config
		if err := localV.Unmarshal(&localConfig); err == nil {
			// Merge hooks
			AppConfig.Hooks.Add = append(AppConfig.Hooks.Add, localConfig.Hooks.Add...)
			AppConfig.Hooks.RM = append(AppConfig.Hooks.RM, localConfig.Hooks.RM...)
			// Merge ignore patterns
			AppConfig.Ignore = append(AppConfig.Ignore, localConfig.Ignore...)
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

			// Fix case for local env
			if localPath != "" {
				if err := loadEnvCasePreserved(localPath); err != nil {
					logger.Warn("failed to restore environment variable case in local config: %v", err)
				}
			}
		} else if ConfigError == nil {
			ConfigError = logger.Errorf("unmarshaling local config: %v", err)
		}
	}

	// 3. Override with --config if provided
	if CfgFile != "" {
		explicitV := viper.New()
		explicitV.SetConfigFile(CfgFile)
		if err := explicitV.ReadInConfig(); err != nil {
			ConfigError = logger.Errorf("reading explicit config: %v", err)
		} else {
			// Full override if specific config is provided
			if err := explicitV.Unmarshal(&AppConfig); err == nil {
				// Fix case for explicit env
				if err := loadEnvCasePreserved(CfgFile); err != nil {
					logger.Warn("failed to restore environment variable case in explicit config: %v", err)
				}
			} else if ConfigError == nil {
				ConfigError = logger.Errorf("unmarshaling explicit config: %v", err)
			}
		}
	}
}

// loadEnvCasePreserved re-reads the config file using yaml.v3 to preserve case in env map
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

	if envRaw, ok := raw["env"]; ok {
		if envMap, ok := envRaw.(map[string]interface{}); ok {
			if AppConfig.Env == nil {
				AppConfig.Env = make(map[string]string)
			}
			for k, v := range envMap {
				if val, ok := v.(string); ok {
					// Remove lowercased duplicate from Viper unmarshal
					lowerK := strings.ToLower(k)
					if _, exists := AppConfig.Env[lowerK]; exists && lowerK != k {
						delete(AppConfig.Env, lowerK)
					}
					AppConfig.Env[k] = val
				}
			}
		}
	}
	return nil
}

// LoadPortsFromPath loads only the ports configuration from a git-wt.config.yml/yaml in the given directory
func LoadPortsFromPath(root string) (map[string]int, error) {
	v := viper.New()
	v.AddConfigPath(root)
	v.SetConfigName("git-wt.config")
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, logger.Errorf("reading config from path %s: %v", root, err)
	}

	var cfg struct {
		Ports map[string]int `mapstructure:"ports"`
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, logger.Errorf("unmarshaling config from path %s: %v", root, err)
	}

	return cfg.Ports, nil
}
