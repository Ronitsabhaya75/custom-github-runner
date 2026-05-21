package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeCacheKey(t *testing.T) {
	// Create a temp workspace directory for testing
	tempDir, err := os.MkdirTemp("", "runner-test-workspace-*")
	if err != nil {
		t.Fatalf("failed to create temp test directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Case 1: No lock files in workspace
	strategy, err := ComputeCacheKey(tempDir, Node)
	if err != nil {
		t.Errorf("unexpected error calculating cache key: %v", err)
	}
	if strategy != nil {
		t.Errorf("expected no cache strategy for empty workspace, got: %+v", strategy)
	}

	// Case 2: Node package-lock.json exists in workspace
	lockContent := []byte(`{"name": "demo", "version": "1.0.0", "dependencies": {}}`)
	lockPath := filepath.Join(tempDir, "package-lock.json")
	if err := os.WriteFile(lockPath, lockContent, 0644); err != nil {
		t.Fatalf("failed to write mock lock file: %v", err)
	}

	strategy, err = ComputeCacheKey(tempDir, Node)
	if err != nil {
		t.Fatalf("unexpected error calculating cache key: %v", err)
	}
	if strategy == nil {
		t.Fatal("expected cache strategy to be generated, got nil")
	}

	if strategy.Language != Node {
		t.Errorf("expected strategy language to be node, got %s", strategy.Language)
	}
	if strategy.CacheDir != "/root/.npm" {
		t.Errorf("expected cache directory to be /root/.npm, got %s", strategy.CacheDir)
	}
	if strategy.CacheKey == "" {
		t.Error("expected non-empty cache key")
	}
	if len(strategy.LockFiles) != 1 || strategy.LockFiles[0] != "package-lock.json" {
		t.Errorf("expected lock files to list package-lock.json, got: %v", strategy.LockFiles)
	}
}

func TestRecommendResources(t *testing.T) {
	tests := []struct {
		name          string
		language      Language
		commands      int
		expectedMem   int64
		expectedCPU   int64
		expectedDisk  int
	}{
		{
			name:         "Python low steps",
			language:     Python,
			commands:     3,
			expectedMem:  128,
			expectedCPU:  500,
			expectedDisk: 1,
		},
		{
			name:         "Python high steps scaling",
			language:     Python,
			commands:     8,
			expectedMem:  256, // scaled * 2
			expectedCPU:  500,
			expectedDisk: 1,
		},
		{
			name:         "Rust heavy compiler",
			language:     Rust,
			commands:     3,
			expectedMem:  512,
			expectedCPU:  2000, // 2 cores allocated
			expectedDisk: 3,
		},
		{
			name:         "Java enterprise runner",
			language:     Java,
			commands:     4,
			expectedMem:  512,
			expectedCPU:  2000,
			expectedDisk: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := RecommendResources(tt.language, tt.commands)
			if recs.MemoryMB != tt.expectedMem {
				t.Errorf("expected MemoryMB %d, got %d", tt.expectedMem, recs.MemoryMB)
			}
			if recs.CPUMillis != tt.expectedCPU {
				t.Errorf("expected CPUMillis %d, got %d", tt.expectedCPU, recs.CPUMillis)
			}
			if recs.StorageGB != tt.expectedDisk {
				t.Errorf("expected StorageGB %d, got %d", tt.expectedDisk, recs.StorageGB)
			}
		})
	}
}
