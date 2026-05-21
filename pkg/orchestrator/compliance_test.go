//go:build integration

package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ronitsabhaya/k8s-github-runner/pkg/sandbox"
	corev1 "k8s.io/api/core/v1"
)

// TestToolchainAndSecurityCompliance runs a comprehensive suite of verification tests
// directly inside the sandboxed container, mimicking the toolchain and security checks
// used in github.com/actions/runner-images.
func TestToolchainAndSecurityCompliance(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to resolve user home directory: %v", err)
	}

	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("Skipping live compliance test: Kubeconfig not found")
	}

	orchestrator, err := NewRunnerOrchestrator(kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to initialize orchestrator: %v", err)
	}

	if orchestrator.DryRun {
		t.Skip("Skipping live compliance test: switched to DryRun simulation mode")
	}

	tests := []struct {
		name       string
		image      string
		lang       sandbox.Language
		commands   []string
		assertions func(t *testing.T, logs string)
	}{
		{
			name:  "Filesystem Isolation (ReadOnlyRootFS)",
			image: "alpine:latest",
			lang:  sandbox.Lightweight,
			commands: []string{
				"echo 'Checking write to root...'",
				"touch /root-write-test.txt 2>&1 || echo 'ROOT_WRITE_BLOCKED'",
				"echo 'Checking write to workspace...'",
				"touch /workspace/workspace-write-test.txt && echo 'WORKSPACE_WRITE_ALLOWED'",
			},
			assertions: func(t *testing.T, logs string) {
				if !strings.Contains(logs, "ROOT_WRITE_BLOCKED") {
					t.Error("Security failure: Root filesystem was not read-only!")
				}
				if !strings.Contains(logs, "WORKSPACE_WRITE_ALLOWED") {
					t.Error("Failure: Writable workspace volume was unreachable!")
				}
			},
		},
		{
			name:  "User Privilege Isolation (Non-Root User)",
			image: "alpine:latest",
			lang:  sandbox.Lightweight,
			commands: []string{
				"echo \"Current User: $(whoami)\"",
				"echo \"UID: $(id -u)\"",
				"id -u | grep -q '1000' && echo 'NON_ROOT_CONFIRMED' || echo 'ROOT_DETECTED'",
			},
			assertions: func(t *testing.T, logs string) {
				if strings.Contains(logs, "ROOT_DETECTED") {
					t.Error("Security failure: container executed as root user!")
				}
				if !strings.Contains(logs, "NON_ROOT_CONFIRMED") {
					t.Error("Security failure: container UID was not restricted to non-root 1000!")
				}
			},
		},
		{
			name:  "Python Toolchain Verification",
			image: "python:3.12-alpine",
			lang:  sandbox.Python,
			commands: []string{
				"python3 --version",
				"pip --version",
				"python3 -c 'import sys; print(\"PYTHON_RUNTIME_OK\")'",
			},
			assertions: func(t *testing.T, logs string) {
				if !strings.Contains(logs, "Python 3.12") {
					t.Error("Toolchain failure: Python 3.12 was not present!")
				}
				if !strings.Contains(logs, "PYTHON_RUNTIME_OK") {
					t.Error("Toolchain failure: Python script execution failed!")
				}
			},
		},
		{
			name:  "NodeJS Toolchain Verification",
			image: "node:20-alpine",
			lang:  sandbox.Node,
			commands: []string{
				"node --version",
				"npm --version",
				"node -e 'console.log(\"NODE_RUNTIME_OK\")'",
			},
			assertions: func(t *testing.T, logs string) {
				if !strings.Contains(logs, "v20.") {
					t.Error("Toolchain failure: Node 20 was not present!")
				}
				if !strings.Contains(logs, "NODE_RUNTIME_OK") {
					t.Error("Toolchain failure: Node script execution failed!")
				}
			},
		},
		{
			name:  "Network Isolation Verification (Blocked)",
			image: "alpine:latest",
			lang:  sandbox.Lightweight,
			commands: []string{
				"echo 'Attempting egress call to Cloud Metadata Server...'",
				"wget -T 3 -O - http://169.254.169.254 2>&1 || echo 'METADATA_EGRESS_BLOCKED'",
			},
			assertions: func(t *testing.T, logs string) {
				if !strings.Contains(logs, "METADATA_EGRESS_BLOCKED") && !strings.Contains(logs, "bad address") {
					t.Error("Security failure: Pod was able to bypass egress isolation rules!")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &sandbox.Policy{
				ReadOnlyFS:   true,
				AllowNetwork: false, // Network policies block all traffic
				MaxMemoryMB:  256,
			}

			job := Job{
				ID:        fmt.Sprintf("compliance-%s", strings.ToLower(strings.ReplaceAll(tt.name, " ", "-"))),
				Namespace: "default",
				Image:     tt.image,
				Commands:  tt.commands,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			// Capture log output via custom buffer or stdout redirection
			// For testing, we stream to a local reader
			t.Logf("Scheduling compliance check pod for: %s", tt.name)
			
			// Custom log capturing via direct client-go integration
			err := orchestrator.ScheduleJob(ctx, job, policy)
			if err != nil {
				t.Fatalf("job execution failed: %v", err)
			}

			// Read pod logs to assert
			req := orchestrator.clientset.CoreV1().Pods(job.Namespace).GetLogs("runner-job-"+job.ID, &corev1.PodLogOptions{})
			stream, err := req.Stream(ctx)
			if err != nil {
				t.Fatalf("failed to retrieve execution logs: %v", err)
			}
			defer stream.Close()

			var buf strings.Builder
			_, _ = io.Copy(&buf, stream)
			logString := buf.String()

			t.Logf("Checking compliance logs:\n%s", logString)
			tt.assertions(t, logString)
		})
	}
}
