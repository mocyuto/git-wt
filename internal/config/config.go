package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/mocyuto/git-wt/internal/git"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	CfgFile string
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
			// Ignore error if global config is not found
		}
	}

	// Unmarshal global config into AppConfig
	if err := v.Unmarshal(&AppConfig); err != nil {
		fmt.Printf("Error unmarshaling global config: %v\n", err)
	}

	// Fix case for global env if loaded
	if globalPath != "" {
		if err := loadEnvCasePreserved(globalPath); err != nil {
			fmt.Printf("Warning: failed to restore environment variable case in global config: %v\n", err)
		}
	}

	// 2. Load local config
	gitRoot, _ := git.GetGitRoot()
	// ... (omitting middle part for clarity, will use AllowMultiple or specific chunks)
	if gitRoot != "" {
		localV := viper.New()
		localV.AddConfigPath(gitRoot)
		localV.SetConfigName("git-wt.config")
		localV.SetConfigType("yaml")

		var localPath string
		if err := localV.ReadInConfig(); err == nil {
			localPath = localV.ConfigFileUsed()
		} else {
			// Try hidden config file
			fmt.Println("Local config load failed", err)
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
					fmt.Printf("Warning: failed to restore environment variable case in local config: %v\n", err)
				}
			}
		} else {
			fmt.Printf("Error unmarshaling local config: %v\n", err)
		}
	}

	// 3. Override with --config if provided
	if CfgFile != "" {
		explicitV := viper.New()
		explicitV.SetConfigFile(CfgFile)
		if err := explicitV.ReadInConfig(); err == nil {
			// Full override if specific config is provided
			if err := explicitV.Unmarshal(&AppConfig); err == nil {
				// Fix case for explicit env
				if err := loadEnvCasePreserved(CfgFile); err != nil {
					fmt.Printf("Warning: failed to restore environment variable case in explicit config: %v\n", err)
				}
			} else {
				fmt.Printf("Error unmarshaling explicit config: %v\n", err)
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
