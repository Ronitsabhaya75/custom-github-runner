package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/ronitsabhaya/k8s-github-runner/pkg/github"
	"github.com/ronitsabhaya/k8s-github-runner/pkg/orchestrator"
	"github.com/ronitsabhaya/k8s-github-runner/pkg/sandbox"
)

func main() {
	// CLI flags
	registerCmd := flag.Bool("register", false, "Register the runner with GitHub Actions")
	listenCmd := flag.Bool("listen", false, "Start the runner daemon polling for jobs")
	simulationFlag := flag.Bool("simulation", false, "Force runner to operate in Simulation Mode")

	// Registration specific flags
	tokenFlag := flag.String("token", "", "GitHub Personal Access Token (PAT) / Registration Token")
	ownerFlag := flag.String("owner", "", "GitHub Repository Owner (username or organization)")
	repoFlag := flag.String("repo", "", "GitHub Repository Name")
	nameFlag := flag.String("name", "custom-k8s-runner", "Custom runner identification name")

	flag.Parse()

	configPath := ".runner-config.json"

	if *registerCmd {
		handleRegistration(configPath, *tokenFlag, *ownerFlag, *repoFlag, *nameFlag)
		return
	}

	if *listenCmd {
		handleListening(configPath, *simulationFlag)
		return
	}

	// Default flow: Run comprehensive multi-language pipeline demonstration
	runStandardDemo()
}

func handleRegistration(configPath, token, owner, repo, name string) {
	fmt.Println("\033[1;36m[Register] Initializing registration flow...\033[0m")

	if token == "" || owner == "" || repo == "" {
		fmt.Println("\033[1;31m❌ Registration error: --token, --owner, and --repo flags are required.\033[0m")
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	client := github.NewClient(ctx, token, owner, repo)

	// Fetch validation registration token from GitHub actions queue to assert authorization
	regToken, err := client.GetRegistrationToken(ctx)
	if err != nil {
		fmt.Printf("\033[1;33m⚠️  Warning: could not verify token with GitHub API: %v\033[0m\n", err)
		fmt.Println("\033[1;35m[Register] Continuing offline registration with provided credentials...\033[0m")
	} else {
		fmt.Printf("\033[1;32m✅ Successfully validated token with GitHub Actions API! Registration token: %s...\033[0m\n", regToken[:8])
	}

	cfg := &github.RunnerConfig{
		Token:    token,
		Owner:    owner,
		Repo:     repo,
		Name:     name,
		IsConfig: true,
	}

	if err := github.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("\033[1;31m❌ Failed to save runner configuration: %v\033[0m\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("\033[1;32m╔══════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Printf("\033[1;32m║  🎉 Runner '%s' Successfully Registered!           ║\033[0m\n", name)
	fmt.Printf("\033[1;32m║  Target: https://github.com/%-32s ║\033[0m\n", owner+"/"+repo)
	fmt.Println("\033[1;32m╚══════════════════════════════════════════════════════════════╝\033[0m")
	fmt.Println("\033[37mTo start polling for jobs, execute: runner --listen\033[0m")
}

func handleListening(configPath string, forceSimulation bool) {
	fmt.Println("\033[1;36m[Listen] Initializing runner listener daemon...\033[0m")

	cfg, err := github.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("\033[1;31m❌ Access error: %v. Please register the runner first using: runner --register\033[0m\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	client := github.NewClient(ctx, cfg.Token, cfg.Owner, cfg.Repo)

	homeDir, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(homeDir, ".kube", "config")

	engine, err := orchestrator.NewRunnerOrchestrator(kubeconfig)
	if err != nil {
		fmt.Printf("Orchestrator error: %v\n", err)
		os.Exit(1)
	}

	if forceSimulation {
		engine.DryRun = true
	}

	if engine.DryRun {
		fmt.Println("\033[1;33m📡 Runner daemon active in high-fidelity SIMULATION mode.\033[0m")
	} else {
		fmt.Println("\033[1;32m📡 Runner daemon connected to live Kubernetes cluster!\033[0m")
	}

	fmt.Printf("Polling Actions queue for \033[1;37mhttps://github.com/%s/%s\033[0m every 5s...\n", cfg.Owner, cfg.Repo)

	// Build default baseline spec for job limits
	policy := &sandbox.Policy{
		ReadOnlyFS:   true,
		AllowNetwork: true,
		MaxMemoryMB:  512,
	}

	for {
		jobs, err := client.FetchPendingJobs(ctx)
		if err != nil {
			fmt.Printf("\033[31m[Poll Error] Failed to retrieve jobs: %v. Retrying...\033[0m\n", err)
		} else if len(jobs) > 0 {
			fmt.Printf("\033[1;32m🚀 Found %d pending jobs in GitHub Actions queue!\033[0m\n", len(jobs))
			for _, job := range jobs {
				lang := sandbox.DetectLanguageFromCommands(job.Steps)
				image := sandbox.ResolveRuntimeImage(lang)
				
				orchJob := orchestrator.Job{
					ID:        fmt.Sprintf("%s-%d", stringsClean(job.Name), job.ID),
					Namespace: "default",
					Image:     image,
					Commands:  job.Steps,
				}

				fmt.Printf("[Dispatcher] Scheduling job %s using environment %s...\n", orchJob.ID, image)
				if err := engine.ScheduleJob(ctx, orchJob, policy); err != nil {
					fmt.Printf("\033[1;31m❌ Job execution failed: %v\033[0m\n", err)
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func stringsClean(s string) string {
	s = filepath.Base(s)
	s = filepath.Clean(s)
	s = filepath.Join(s)
	s = filepath.Clean(s)
	return s
}

func runStandardDemo() {
	fmt.Println()
	fmt.Println("\033[1;97m  ╔══════════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;97m  ║\033[0m  \033[1;36m🚀 Sandboxed Kubernetes GitHub Runner v0.1.0\033[0m               \033[1;97m║\033[0m")
	fmt.Println("\033[1;97m  ║\033[0m  \033[37m   Lightweight • Secure • Every Language\033[0m                  \033[1;97m║\033[0m")
	fmt.Println("\033[1;97m  ╚══════════════════════════════════════════════════════════════╝\033[0m")
	fmt.Println()

	langs := sandbox.ListSupportedLanguages()
	fmt.Printf("\033[1;33m📋 Registered Languages: %d\033[0m\n", len(langs))
	fmt.Println()

	bundleDir, err := os.MkdirTemp("", "oci-bundle-*")
	if err != nil {
		fmt.Printf("Failed to create temporary OCI bundle directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(bundleDir)

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

	policy, err := sandbox.LoadPolicy(bundleDir)
	if err != nil {
		fmt.Printf("Failed to load OCI sandbox policy: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\033[37m[Init] Sandbox Policy: ReadOnlyFS=%t, MemoryLimit=%dMB, Network=%t\033[0m\n",
		policy.ReadOnlyFS, policy.MaxMemoryMB, policy.AllowNetwork)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error resolving user home directory: %v\n", err)
		os.Exit(1)
	}
	kubeconfig := filepath.Join(homeDir, ".kube", "config")

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

	fmt.Println()
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[1;36m  Running 10-Language CI Pipeline Demo\033[0m")
	fmt.Println("\033[1;97m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println()

	ctx := context.Background()
	overallStart := time.Now()
	results := engine.ExecuteMultiLanguageDemo(ctx, policy)

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
