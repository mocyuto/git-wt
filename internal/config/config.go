package config

import (
	"io"
	"maps"
	"os"
	"path/filepath"
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
	globalPath, err := GetGlobalConfigPath()
	if err == nil {
		configDir := filepath.Dir(globalPath)
		v.AddConfigPath(configDir)
		v.SetConfigName(GlobalConfigName)
		v.SetConfigType(GlobalConfigExt)

		// Create default global config if it doesn't exist
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
				if err := loadEnvCasePreserved(localPath); err != nil {
					logger.Warn("failed to restore environment variable case in local config: %v", err)
				}
			} else if ConfigError == nil {
				ConfigError = logger.Errorf("unmarshaling local config: %v", err)
			}
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
