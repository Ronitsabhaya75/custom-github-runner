package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ronitsabhaya/k8s-github-runner/pkg/sandbox"
)

// JobResult holds the outcome of a single job execution
type JobResult struct {
	JobID      string
	Language   sandbox.Language
	Image      string
	Success    bool
	Duration   time.Duration
	StepCount  int
	MemoryUsed int64 // in MB
	CacheHit   bool
	Error      string
}

// executeSimulatedJob simulates the high-fidelity sandboxed execution of CI tasks on the terminal
func (o *RunnerOrchestrator) executeSimulatedJob(ctx context.Context, job Job, policy *sandbox.Policy) error {
	podName := fmt.Sprintf("runner-job-%s", job.ID)
	lang := sandbox.DetectLanguageFromCommands(job.Commands)
	runtimeCfg := sandbox.ResolveRuntimeConfig(lang)
	resources := sandbox.RecommendResources(lang, len(job.Commands))

	startTime := time.Now()

	fmt.Printf("\n\033[1;97m╔══════════════════════════════════════════════════════════════╗\033[0m\n")
	fmt.Printf("\033[1;97m║\033[0m  \033[1;36m⚡ Pod: %-52s\033[1;97m║\033[0m\n", podName)
	fmt.Printf("\033[1;97m║\033[0m  \033[1;33m🌐 Namespace: %-46s\033[1;97m║\033[0m\n", job.Namespace)
	fmt.Printf("\033[1;97m╚══════════════════════════════════════════════════════════════╝\033[0m\n")
	time.Sleep(150 * time.Millisecond)

	// Phase 1: Security Context Translation
	fmt.Printf("\n\033[1;35m┌─ 🔒 Security Context\033[0m\n")
	printKV("ReadOnlyRootFilesystem", fmt.Sprintf("%t", policy.ReadOnlyFS), "32")
	printKV("RunAsNonRoot", "true", "32")
	printKV("RunAsUser", "1000 (non-root worker)", "32")
	printKV("AllowPrivilegeEscalation", "false", "31")
	printKV("Capabilities.Drop", "[ALL]", "31")
	time.Sleep(150 * time.Millisecond)

	// Phase 2: Resource Allocation (auto-tuned)
	fmt.Printf("\n\033[1;35m┌─ 📊 Resource Allocation (auto-tuned for %s)\033[0m\n", lang)
	printKV("Memory Limit", fmt.Sprintf("%dMi", resources.MemoryMB), "36")
	printKV("CPU Request", fmt.Sprintf("%dm (%.1f cores)", resources.CPUMillis, float64(resources.CPUMillis)/1000), "36")
	printKV("Ephemeral Storage", fmt.Sprintf("%dGi", resources.StorageGB), "36")
	printKV("Image", runtimeCfg.Image, "34")
	printKV("Image Size", fmt.Sprintf("~%dMB", runtimeCfg.SizeMB), "34")
	time.Sleep(150 * time.Millisecond)

	// Phase 3: Network Policy
	fmt.Printf("\n\033[1;35m┌─ 🛡️  Network Policy\033[0m\n")
	if policy.AllowNetwork {
		printKV("Egress", "ALLOW public internet", "32")
		printKV("Blocked", "169.254.169.254 (metadata), 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16", "31")
	} else {
		printKV("Egress", "BLOCKED (air-gapped sandbox)", "31")
	}
	printKV("Ingress", "BLOCKED (zero inbound)", "31")
	time.Sleep(150 * time.Millisecond)

	// Phase 4: Dependency Caching
	fmt.Printf("\n\033[1;35m┌─ 💾 Dependency Cache\033[0m\n")
	if runtimeCfg.CacheDir != "" {
		printKV("Cache Directory", runtimeCfg.CacheDir, "36")
		printKV("Lock Files", strings.Join(runtimeCfg.LockFiles, ", "), "36")
		printKV("Package Restore", runtimeCfg.PackageCmd, "33")
		printKV("Status", "CACHE MISS → installing fresh dependencies", "33")
	} else {
		printKV("Status", "No cache needed for this language", "37")
	}
	time.Sleep(150 * time.Millisecond)

	// Phase 5: Pod Lifecycle
	fmt.Printf("\n\033[1;35m┌─ 🔄 Pod Lifecycle\033[0m\n")
	printKV("State", "PENDING → pulling image", "33")
	time.Sleep(300 * time.Millisecond)
	printKV("State", "RUNNING → container started", "32")
	time.Sleep(100 * time.Millisecond)

	// Phase 6: Command Execution
	fmt.Printf("\n\033[1;35m┌─ 📺 Live Output Stream\033[0m\n")
	for i, cmd := range job.Commands {
		fmt.Printf("│  \033[1;37mStep %d/%d:\033[0m \033[36m$ %s\033[0m\n", i+1, len(job.Commands), cmd)
		time.Sleep(150 * time.Millisecond)

		output := simulateCommandOutput(cmd)
		for _, line := range output {
			fmt.Printf("│  \033[37m%s\033[0m\n", line)
		}
		time.Sleep(80 * time.Millisecond)
	}

	// Phase 7: Cleanup
	elapsed := time.Since(startTime)
	fmt.Printf("\n\033[1;35m┌─ 🧹 Cleanup\033[0m\n")
	printKV("NetworkPolicy", fmt.Sprintf("isolate-%s → deleted", podName), "32")
	printKV("Pod", fmt.Sprintf("%s → terminated", podName), "32")
	printKV("Duration", elapsed.Round(time.Millisecond).String(), "36")

	fmt.Printf("\n\033[1;32m✅ Job %s completed successfully\033[0m\n", job.ID)

	return nil
}

// ExecuteMultiLanguageDemo runs a comprehensive demo across all tiers of languages
func (o *RunnerOrchestrator) ExecuteMultiLanguageDemo(ctx context.Context, policy *sandbox.Policy) []JobResult {
	demos := []Job{
		{ID: "python-ci", Namespace: "default", Commands: []string{
			"python3 --version",
			"pip install flask pytest",
			"python3 -c \"import flask; print(f'Flask {flask.__version__} loaded')\"",
			"pytest --version",
		}},
		{ID: "node-ci", Namespace: "default", Commands: []string{
			"node --version",
			"npm init -y",
			"npm install express",
			"node -e \"const express = require('express'); console.log('Express loaded')\"",
		}},
		{ID: "go-ci", Namespace: "default", Commands: []string{
			"go version",
			"go mod init demo",
			"go build ./...",
			"go test ./...",
		}},
		{ID: "rust-ci", Namespace: "default", Commands: []string{
			"rustc --version",
			"cargo init .",
			"cargo build --release",
			"cargo test",
		}},
		{ID: "java-ci", Namespace: "default", Commands: []string{
			"java --version",
			"javac Main.java",
			"java Main",
			"mvn test",
		}},
		{ID: "cpp-ci", Namespace: "default", Commands: []string{
			"g++ --version",
			"cmake --version",
			"g++ -O2 -o main main.cpp",
			"./main",
		}},
		{ID: "ruby-ci", Namespace: "default", Commands: []string{
			"ruby --version",
			"gem install bundler",
			"bundle install",
			"rake test",
		}},
		{ID: "dotnet-ci", Namespace: "default", Commands: []string{
			"dotnet --version",
			"dotnet new console",
			"dotnet build",
			"dotnet test",
		}},
		{ID: "elixir-ci", Namespace: "default", Commands: []string{
			"elixir --version",
			"mix new demo",
			"mix compile",
			"mix test",
		}},
		{ID: "terraform-ci", Namespace: "default", Commands: []string{
			"terraform version",
			"terraform init",
			"terraform validate",
			"terraform plan",
		}},
	}

	var results []JobResult
	var mu sync.Mutex

	for _, job := range demos {
		lang := sandbox.DetectLanguageFromCommands(job.Commands)
		job.Image = sandbox.ResolveRuntimeImage(lang)

		start := time.Now()
		err := o.ScheduleJob(ctx, job, policy)
		elapsed := time.Since(start)

		result := JobResult{
			JobID:    job.ID,
			Language: lang,
			Image:    job.Image,
			Success:  err == nil,
			Duration: elapsed,
			StepCount: len(job.Commands),
		}
		if err != nil {
			result.Error = err.Error()
		}

		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}

	return results
}

// printKV prints a styled key-value pair with ANSI color for the value
func printKV(key, value, colorCode string) {
	padding := 28 - len(key)
	if padding < 1 {
		padding = 1
	}
	fmt.Printf("│  %-*s\033[%sm%s\033[0m\n", 28, key+":", colorCode, value)
}

// simulateCommandOutput returns realistic stdout for known commands
func simulateCommandOutput(cmd string) []string {
	c := strings.ToLower(cmd)

	// Version commands
	switch {
	case strings.HasPrefix(c, "python3 --version") || strings.HasPrefix(c, "python --version"):
		return []string{"Python 3.12.3"}
	case strings.HasPrefix(c, "node --version"):
		return []string{"v20.11.0"}
	case strings.HasPrefix(c, "go version"):
		return []string{"go version go1.22.2 linux/amd64"}
	case strings.HasPrefix(c, "rustc --version"):
		return []string{"rustc 1.77.0 (aedd173a2 2024-03-17)"}
	case strings.HasPrefix(c, "java --version"):
		return []string{"openjdk 21.0.2 2024-01-16 LTS", "OpenJDK Runtime Environment Temurin-21.0.2+13 (build 21.0.2+13-LTS)"}
	case strings.HasPrefix(c, "ruby --version"):
		return []string{"ruby 3.3.0 (2024-12-25 revision 5124f9ac75) [aarch64-linux]"}
	case strings.HasPrefix(c, "g++ --version"):
		return []string{"g++ (Alpine 13.2.1_git20240309) 13.2.1 20240309"}
	case strings.HasPrefix(c, "dotnet --version"):
		return []string{"8.0.201"}
	case strings.HasPrefix(c, "elixir --version"):
		return []string{"Erlang/OTP 26 [erts-14.2.2]", "Elixir 1.16.1 (compiled with Erlang/OTP 26)"}
	case strings.HasPrefix(c, "terraform version"):
		return []string{"Terraform v1.7.4", "on linux_amd64"}
	case strings.HasPrefix(c, "cmake --version"):
		return []string{"cmake version 3.28.3"}
	case strings.HasPrefix(c, "go env"):
		return []string{"linux", "amd64"}

	// Build / Install commands
	case strings.Contains(c, "pip install"):
		return []string{"Collecting packages...", "Installing collected packages...", "Successfully installed all packages"}
	case strings.Contains(c, "npm install") || strings.Contains(c, "npm init"):
		return []string{"added 57 packages in 2.1s"}
	case strings.Contains(c, "cargo build"):
		return []string{"   Compiling demo v0.1.0", "    Finished release [optimized] target(s) in 1.23s"}
	case strings.Contains(c, "cargo init"):
		return []string{"     Created binary (application) package"}
	case strings.Contains(c, "go mod init"):
		return []string{"go: creating new go.mod: module demo"}
	case strings.Contains(c, "go build"):
		return []string{""}
	case strings.Contains(c, "gem install"):
		return []string{"Successfully installed bundler-2.5.6", "1 gem installed"}
	case strings.Contains(c, "bundle install"):
		return []string{"Fetching gem metadata...", "Bundle complete! 3 Gemfile dependencies, 12 gems now installed."}
	case strings.Contains(c, "dotnet new"):
		return []string{"The template \"Console App\" was created successfully."}
	case strings.Contains(c, "dotnet build"):
		return []string{"Build succeeded.", "    0 Warning(s)", "    0 Error(s)"}
	case strings.Contains(c, "mix new"):
		return []string{"* creating README.md", "* creating lib/demo.ex", "* creating test/demo_test.exs"}
	case strings.Contains(c, "mix compile"):
		return []string{"Compiling 1 file (.ex)", "Generated demo app"}
	case strings.Contains(c, "terraform init"):
		return []string{"Terraform has been successfully initialized!"}
	case strings.Contains(c, "terraform validate"):
		return []string{"Success! The configuration is valid."}

	// Test commands
	case strings.Contains(c, "pytest"):
		return []string{"===== 12 passed in 0.43s ====="}
	case strings.Contains(c, "go test"):
		return []string{"ok  \tdemo\t0.003s"}
	case strings.Contains(c, "cargo test"):
		return []string{"running 3 tests", "test result: ok. 3 passed; 0 failed; 0 ignored"}
	case strings.Contains(c, "mvn test") || strings.Contains(c, "rake test"):
		return []string{"Tests run: 5, Failures: 0, Errors: 0, Skipped: 0", "BUILD SUCCESS"}
	case strings.Contains(c, "dotnet test"):
		return []string{"Passed!  - Failed: 0, Passed: 3, Skipped: 0, Total: 3"}
	case strings.Contains(c, "mix test"):
		return []string{"...", "3 tests, 0 failures"}
	case strings.Contains(c, "terraform plan"):
		return []string{"No changes. Your infrastructure matches the configuration."}
	}

	// Generic fallback
	return []string{"\033[32m✓ completed\033[0m"}
}
