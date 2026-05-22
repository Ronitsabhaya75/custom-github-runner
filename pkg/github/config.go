package github

import (
	"encoding/json"
	"fmt"
	"os"
)

// RunnerConfig holds the registration parameters for the self-hosted runner
type RunnerConfig struct {
	Token    string `json:"token"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Name     string `json:"name"`
	IsConfig bool   `json:"is_configured"`
}

// LoadConfig reads the runner settings from the local file system
func LoadConfig(path string) (*RunnerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read runner config: %w", err)
	}

	var cfg RunnerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal runner config: %w", err)
	}

	return &cfg, nil
}

// SaveConfig persists the runner settings to the local file system
func SaveConfig(path string, cfg *RunnerConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal runner config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write runner config: %w", err)
	}

	return nil
}
