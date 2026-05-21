package sandbox

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CacheStrategy describes how to persist and restore dependency caches between runs
type CacheStrategy struct {
	Language   Language
	CacheKey   string   // SHA256 hash of lock files for cache invalidation
	CacheDir   string   // Container path to cache (e.g., /root/.cache/pip)
	VolumeName string   // K8s PVC or hostPath volume name
	LockFiles  []string // Which files were used to compute the key
	HitRate    float64  // Estimated cache hit probability (for telemetry)
}

// ComputeCacheKey generates a deterministic SHA256 cache key from dependency lock files
// This is equivalent to what GitHub Actions' actions/cache does, but built directly
// into the runner instead of requiring a separate Action step.
func ComputeCacheKey(workspaceDir string, lang Language) (*CacheStrategy, error) {
	cfg := ResolveRuntimeConfig(lang)
	if cfg.CacheDir == "" || len(cfg.LockFiles) == 0 {
		return nil, nil // Language doesn't benefit from caching
	}

	hasher := sha256.New()
	hasher.Write([]byte(string(lang))) // Salt with language name
	foundFiles := []string{}

	for _, pattern := range cfg.LockFiles {
		matches, err := filepath.Glob(filepath.Join(workspaceDir, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			hasher.Write(data)
			foundFiles = append(foundFiles, filepath.Base(match))
		}
	}

	if len(foundFiles) == 0 {
		return nil, nil // No lock files found, skip caching
	}

	cacheKey := fmt.Sprintf("%s-%x", lang, hasher.Sum(nil)[:12])

	return &CacheStrategy{
		Language:   lang,
		CacheKey:   cacheKey,
		CacheDir:   cfg.CacheDir,
		VolumeName: fmt.Sprintf("cache-%s", sanitizeVolumeName(cacheKey)),
		LockFiles:  foundFiles,
		HitRate:    0.0, // Populated at runtime from telemetry
	}, nil
}

// sanitizeVolumeName converts a cache key into a valid K8s volume name
func sanitizeVolumeName(key string) string {
	cleaned := strings.ToLower(key)
	cleaned = strings.ReplaceAll(cleaned, "_", "-")
	// K8s names must be <= 63 chars, start/end with alphanumeric
	if len(cleaned) > 50 {
		cleaned = cleaned[:50]
	}
	return cleaned
}

// ResourceRecommendation dynamically adjusts pod resource limits based on language profile
type ResourceRecommendation struct {
	MemoryMB    int64
	CPUMillis   int64 // 1000m = 1 CPU core
	NeedsGPU    bool
	StorageGB   int
}

// RecommendResources returns optimal resource allocation based on language + job complexity
func RecommendResources(lang Language, commandCount int) ResourceRecommendation {
	cfg := ResolveRuntimeConfig(lang)
	baseMem := cfg.MinMemoryMB

	// Scale memory up based on number of parallel steps
	if commandCount > 5 {
		baseMem = baseMem * 2
	}
	if commandCount > 15 {
		baseMem = baseMem * 3
	}

	// CPU allocation: compiled languages get more cores
	cpuMillis := int64(500) // 0.5 cores default
	switch lang {
	case Rust, Cpp, Java, Scala, Haskell, Swift:
		cpuMillis = 2000 // 2 full cores for heavy compilers
	case Go, Kotlin, Csharp:
		cpuMillis = 1000 // 1 core
	case Node, Python, Ruby, PHP:
		cpuMillis = 500 // 0.5 cores (interpreted, less CPU-bound)
	}

	// Storage: some ecosystems pull massive dependency trees
	storageGB := 1
	switch lang {
	case Java, Scala, Kotlin, Haskell, Swift:
		storageGB = 5
	case Rust, Cpp, Csharp:
		storageGB = 3
	case Node:
		storageGB = 2 // node_modules can be large
	}

	return ResourceRecommendation{
		MemoryMB:  baseMem,
		CPUMillis: cpuMillis,
		NeedsGPU:  false,
		StorageGB: storageGB,
	}
}
