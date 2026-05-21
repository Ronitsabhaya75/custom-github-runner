package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// Policy represents the sandboxing policy extracted from the official OCI spec
type Policy struct {
	AllowedCommands []string `json:"allowedCommands"`
	AllowNetwork    bool     `json:"allowNetwork"`
	ReadOnlyFS      bool     `json:"readOnlyFS"`
	MaxMemoryMB     int64    `json:"maxMemoryMB"`
}

// LoadPolicy reads the OCI bundle config.json (using the open-source specs.Spec) and extracts the policy
func LoadPolicy(bundlePath string) (*Policy, error) {
	configPath := filepath.Join(bundlePath, "config.json")
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open OCI config.json: %w", err)
	}
	defer configFile.Close()

	var spec specs.Spec
	if err := json.NewDecoder(configFile).Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode official OCI spec: %w", err)
	}

	// Extract standard OCI properties into our high-level policy
	policy := &Policy{
		ReadOnlyFS: spec.Root != nil && spec.Root.Readonly,
	}

	// 1. Extract allowed commands (from OCI process args)
	if spec.Process != nil {
		policy.AllowedCommands = spec.Process.Args
	}

	// 2. Extract network namespace configuration
	policy.AllowNetwork = true
	if spec.Linux != nil {
		for _, ns := range spec.Linux.Namespaces {
			if ns.Type == specs.NetworkNamespace {
				if ns.Path == "" {
					policy.AllowNetwork = false
				}
			}
		}

		// 3. Extract memory limits (convert bytes to MB)
		if spec.Linux.Resources != nil && spec.Linux.Resources.Memory != nil && spec.Linux.Resources.Memory.Limit != nil {
			bytesLimit := *spec.Linux.Resources.Memory.Limit
			if bytesLimit > 0 {
				policy.MaxMemoryMB = bytesLimit / (1024 * 1024)
			}
		}
	}

	return policy, nil
}
