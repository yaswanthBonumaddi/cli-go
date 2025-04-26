package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/godspeedsystems/godspeed-cli/internal/utils"
	"github.com/spf13/viper"
)

// Init initializes the configuration
func Init() error {
	// Set default values
	viper.SetDefault("GITHUB_REPO_URL", "https://github.com/godspeedsystems/godspeed-scaffolding.git")
	viper.SetDefault("GITHUB_REPO_BRANCH", "template")
	viper.SetDefault("DOCKER_REGISTRY_TAGS_VERSION_URL", "https://registry.hub.docker.com/v2/namespaces/godspeedsystems/repositories/gs-node-service/tags?n=5")
	viper.SetDefault("DOCKER_REGISTRY", "godspeedsystems")
	viper.SetDefault("DOCKER_PACKAGE_NAME", "gs-node-service")
	viper.SetDefault("RUN_TESTS", "FALSE")

	// Look for .env file
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Read the configuration file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if no config file is found
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// Auto-read environment variables
	viper.AutomaticEnv()

	return nil
}

// LoadPluginsList loads the plugins list from the embedded asset
func LoadPluginsList() ([]map[string]interface{}, error) {
	// Get the executable path
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	// Path to the plugins list json
	pluginsListPath := filepath.Join(filepath.Dir(execPath), "assets", "plugins_list.json")

	// If the file doesn't exist, use a default empty list
	if !utils.FileExists(pluginsListPath) {
		return []map[string]interface{}{}, nil
	}

	// Read the file
	data, err := os.ReadFile(pluginsListPath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON
	var plugins []map[string]interface{}
	if err := json.Unmarshal(data, &plugins); err != nil {
		return nil, err
	}

	return plugins, nil
}

// SaveGodspeedConfig saves the godspeed configuration to a file
func SaveGodspeedConfig(configPath string, config map[string]interface{}) error {
	// Convert to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0644)
}

// LoadGodspeedConfig loads the godspeed configuration from a file
func LoadGodspeedConfig(configPath string) (map[string]interface{}, error) {
	// Check if file exists
	if !utils.FileExists(configPath) {
		return nil, os.ErrNotExist
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}
