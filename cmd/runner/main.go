package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/ronitsabhaya/k8s-github-runner/pkg/orchestrator"
	"github.com/ronitsabhaya/k8s-github-runner/pkg/sandbox"
)

func main() {
	fmt.Println()
	fmt.Println("\033[1;97m  ╔══════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;97m  ║\033[0m  \033[1;36m🚀 Sandboxed Kubernetes GitHub Runner v0.1.0\033[0m               \033[1;97m║\033[0m")
	fmt.Println("\033[1;97m  ║\033[0m  \033[37m   Lightweight • Secure • Every Language\033[0m                  \033[1;97m║\033[0m")
	fmt.Println("\033[1;97m  ╚══════════════════════════════════════════════════════════════╝\033[0m")
	fmt.Println()

	// Print supported language count
	langs := sandbox.ListSupportedLanguages()
	fmt.Printf("\033[1;33m📋 Registered Languages: %d\033[0m\n", len(langs))
	fmt.Println()

	// 1. Create a temporary OCI bundle directory with an official config.json spec
	bundleDir, err := os.MkdirTemp("", "oci-bundle-*")
	if err != nil {
		fmt.Printf("Failed to create temporary OCI bundle directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(bundleDir)

	// Write mock official specs.Spec
	limitBytes := int64(512 * 1024 * 1024) // 512MB default
	spec := specs.Spec{
		Version: "1.0.0",
		Process: &specs.Process{
			Args: []string{"echo", "whoami", "ls"},
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

	configPath := filepath.Join(bundleDir, "config.json")
	configFile, err := os.Create(configPath)
	if err != nil {
		fmt.Printf("Failed to create mock OCI config.json: %v\n", err)
		os.Exit(1)
	}
	if err := json.NewEncoder(configFile).Encode(spec); err != nil {
		configFile.Close()
		fmt.Printf("Failed to encode mock OCI config.json: %v\n", err)
		os.Exit(1)
	}
	configFile.Close()
	fmt.Printf("\033[37m[Init] OCI bundle spec created at: %s\033[0m\n", configPath)

	// 2. Load policy from OCI bundle (using open-source specs-go)
	policy, err := sandbox.LoadPolicy(bundleDir)
	if err != nil {
		fmt.Printf("Failed to load OCI sandbox policy: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\033[37m[Init] Sandbox Policy: ReadOnlyFS=%t, MemoryLimit=%dMB, Network=%t\033[0m\n",
		policy.ReadOnlyFS, policy.MaxMemoryMB, policy.AllowNetwork)

	// 3. Resolve local kubeconfig for development
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error resolving user home directory: %v\n", err)
		os.Exit(1)
	}
	kubeconfig := filepath.Join(homeDir, ".kube", "config")

	// 4. Initialize Runner Orchestrator
	engine, err := orchestrator.NewRunnerOrchestrator(kubeconfig)
	if err != nil {
		fmt.Printf("Failed to initialize orchestrator: %v\n", err)
		os.Exit(1)
	}
	if engine.DryRun {
		fmt.Printf("\033[33m[Init] Running in Simulation Mode (cluster unreachable)\033[0m\n")
	} else {
		fmt.Printf("\033[32m[Init] Connected to live Kubernetes cluster\033[0m\n")
	}

	// 5. Run comprehensive multi-language demo
	fmt.Println()
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[1;36m  Running 10-Language CI Pipeline Demo\033[0m")
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println()

	ctx := context.Background()
	overallStart := time.Now()
	results := engine.ExecuteMultiLanguageDemo(ctx, policy)

	// 6. Print summary report
	totalDuration := time.Since(overallStart)
	fmt.Println()
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[1;36m  📊 Execution Summary Report\033[0m")
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println()

	passed := 0
	failed := 0
	for _, r := range results {
		status := "\033[32m✅ PASSED\033[0m"
		if !r.Success {
			status = "\033[31m❌ FAILED\033[0m"
			failed++
		} else {
			passed++
		}
		fmt.Printf("  %s  %-20s %-28s %s  (%d steps)\n",
			status, r.JobID, r.Image, r.Duration.Round(time.Millisecond), r.StepCount)
	}

	fmt.Println()
	fmt.Printf("  \033[1;37mTotal Jobs:  %d\033[0m\n", len(results))
	fmt.Printf("  \033[1;32mPassed:      %d\033[0m\n", passed)
	if failed > 0 {
		fmt.Printf("  \033[1;31mFailed:      %d\033[0m\n", failed)
	}
	fmt.Printf("  \033[1;36mTotal Time:  %s\033[0m\n", totalDuration.Round(time.Millisecond))
	fmt.Println()

	// Compare with GitHub hosted runner
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[1;36m  🆚 Comparison: This Runner vs GitHub Hosted Runner\033[0m")
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println()
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Feature", "This Runner", "GitHub Hosted")
	fmt.Printf("  %-30s %-20s %-20s\n", "──────────────────────────────", "────────────────────", "────────────────────")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Runtime", "Native Go (~10MB)", ".NET Core (~100MB+)")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Base Image", "Alpine (~5-85MB)", "Ubuntu (~50-80GB)")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Languages", fmt.Sprintf("%d (auto-detect)", len(langs)), "Pre-installed blob")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Security Isolation", "Per-pod + NetworkPolicy", "Shared VM")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Resource Tuning", "Auto per language", "Fixed 2-core/7GB")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Dependency Caching", "Built-in (lock hash)", "Requires actions/cache")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Cleanup", "Auto pod deletion", "VM persists")
	fmt.Printf("  %-30s \033[32m%-20s\033[0m \033[31m%-20s\033[0m\n", "Metadata Protection", "Blocks 169.254.x.x", "Exposed")
	fmt.Println()
}
