package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	cfgFile string
)

type Config struct {
	Hooks struct {
		Add []string `mapstructure:"add"`
		RM  []string `mapstructure:"rm"`
	} `mapstructure:"hooks"`
	Ignore []string       `mapstructure:"ignore"`
	Ports  map[string]int `mapstructure:"ports"`
}

var AppConfig Config

func initConfig() {
	// Initialize default viper for global/local loading
	v := viper.New()

	// 1. Load global config
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config", "git-wt")
		v.AddConfigPath(configDir)
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Create default global config if it doesn't exist
		configPath := filepath.Join(configDir, "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err == nil {
				v.SetDefault("hooks.add", []string{})
				v.SetDefault("hooks.rm", []string{})
				v.SetDefault("ignore", []string{})
				_ = v.SafeWriteConfigAs(configPath)
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

	// 2. Load local config
	gitRoot, _ := GetGitRoot()
	if gitRoot != "" {
		localV := viper.New()
		localV.AddConfigPath(gitRoot)
		localV.SetConfigName("git-wt.config")
		// Viper will look for .yaml, .yml, .json, etc. if no type is set but we want to be sure

		if err := localV.ReadInConfig(); err == nil {
			var localConfig Config
			if err := localV.Unmarshal(&localConfig); err == nil {
				// Merge hooks
				AppConfig.Hooks.Add = append(AppConfig.Hooks.Add, localConfig.Hooks.Add...)
				AppConfig.Hooks.RM = append(AppConfig.Hooks.RM, localConfig.Hooks.RM...)
				// Merge ignore patterns
				AppConfig.Ignore = append(AppConfig.Ignore, localConfig.Ignore...)
			}
		}
	}

	// 3. Override with --config if provided
	if cfgFile != "" {
		explicitV := viper.New()
		explicitV.SetConfigFile(cfgFile)
		if err := explicitV.ReadInConfig(); err == nil {
			// Full override if specific config is provided
			if err := explicitV.Unmarshal(&AppConfig); err != nil {
				fmt.Printf("Error unmarshaling explicit config: %v\n", err)
			}
		}
	}
}
