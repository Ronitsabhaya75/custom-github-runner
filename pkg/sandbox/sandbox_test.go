package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func TestLoadPolicy(t *testing.T) {
	// Create a temp OCI bundle directory for testing
	tempDir, err := os.MkdirTemp("", "runner-test-oci-*")
	if err != nil {
		t.Fatalf("failed to create temp OCI directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock specs.Spec
	limitBytes := int64(1024 * 1024 * 512) // 512MB
	spec := specs.Spec{
		Version: "1.0.0",
		Process: &specs.Process{
			Args: []string{"echo", "test"},
		},
		Root: &specs.Root{
			Readonly: true,
		},
		Linux: &specs.Linux{
			Namespaces: []specs.LinuxNamespace{
				{Type: specs.NetworkNamespace, Path: ""},
			},
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{
					Limit: &limitBytes,
				},
			},
		},
	}

	// Write mock config.json
	configPath := filepath.Join(tempDir, "config.json")
	configFile, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("failed to create mock config.json: %v", err)
	}
	if err := json.NewEncoder(configFile).Encode(spec); err != nil {
		configFile.Close()
		t.Fatalf("failed to write mock config.json: %v", err)
	}
	configFile.Close()

	// Load and assert
	policy, err := LoadPolicy(tempDir)
	if err != nil {
		t.Fatalf("LoadPolicy failed: %v", err)
	}

	if !policy.ReadOnlyFS {
		t.Error("expected ReadOnlyFS to be true")
	}
	if policy.AllowNetwork {
		t.Error("expected AllowNetwork to be false due to empty NetworkNamespace path")
	}
	if policy.MaxMemoryMB != 512 {
		t.Errorf("expected MaxMemoryMB to be 512, got %d", policy.MaxMemoryMB)
	}
	if len(policy.AllowedCommands) != 2 || policy.AllowedCommands[0] != "echo" {
		t.Errorf("unexpected allowed commands: %v", policy.AllowedCommands)
	}
}
