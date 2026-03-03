package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AnthropicAPIKey   string `yaml:"anthropic_api_key"`
	Model             string `yaml:"model"`
	FirecrawlAPIKey   string `yaml:"firecrawl_api_key"` // Optional - enables web search
}

func Path() (string, error) {
	// Try XDG config location first
	configPath, err := xdg.SearchConfigFile("readctl/config.yaml")
	if err != nil {
		// Use XDG config home or fallback to ~/.config
		configDir := filepath.Join(xdg.ConfigHome, "readctl")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "config.yaml")
	}
	return configPath, nil
}

func Save(cfg *Config) error {
	configPath, err := Path()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func Load() (*Config, error) {
	configPath, err := Path()
	if err != nil {
		return nil, err
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s - please create it with anthropic_api_key and model fields", configPath)
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("anthropic_api_key is required in config.yaml")
	}

	// Set defaults
	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-5-20250929"
	}

	return &cfg, nil
}
