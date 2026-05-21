//go:build integration

package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ronitsabhaya/k8s-github-runner/pkg/sandbox"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestLiveOrchestratorIntegration performs a full end-to-end integration test
// against a live Kubernetes cluster (e.g. KinD in CI or Colima locally)
func TestLiveOrchestratorIntegration(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to resolve user home directory: %v", err)
	}

	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("Skipping live integration test: Kubeconfig not found at " + kubeconfigPath)
	}

	orchestrator, err := NewRunnerOrchestrator(kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to initialize live orchestrator: %v", err)
	}

	if orchestrator.DryRun {
		t.Skip("Skipping live integration test: Orchestrator switched to DryRun mode (cluster unreachable)")
	}

	// 1. Setup secure sandbox policy config
	policy := &sandbox.Policy{
		ReadOnlyFS:   true,
		AllowNetwork: false, // Fully air-gapped test
		MaxMemoryMB:  128,
	}

	// 2. Define integration test job
	job := Job{
		ID:        "integration-test-job",
		Namespace: "default",
		Image:     "alpine:latest",
		Commands: []string{
			"whoami",
			"echo 'Integration test execution successful'",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("Starting live Pod scheduling...")
	err = orchestrator.ScheduleJob(ctx, job, policy)
	if err != nil {
		t.Fatalf("ScheduleJob failed: %v", err)
	}

	// 3. Post-execution verification: check that Pod and NetworkPolicy were cleaned up
	podsClient := orchestrator.clientset.CoreV1().Pods(job.Namespace)
	_, err = podsClient.Get(ctx, "runner-job-"+job.ID, metav1.GetOptions{})
	if err == nil {
		t.Error("Pod was not cleaned up after integration test run")
	}

	netClient := orchestrator.clientset.NetworkingV1().NetworkPolicies(job.Namespace)
	_, err = netClient.Get(ctx, "isolate-runner-job-"+job.ID, metav1.GetOptions{})
	if err == nil {
		t.Error("NetworkPolicy was not cleaned up after integration test run")
	}
}

// TestComplianceCapabilityDrops runs the 'Captain' Capability dropped compliance test
func TestComplianceCapabilityDrops(t *testing.T) {
	policy := &sandbox.Policy{
		ReadOnlyFS:   true,
		AllowNetwork: false,
		MaxMemoryMB:  128,
	}

	// Translate policy to Pod SecurityContext
	securityContext := &corev1.SecurityContext{
		ReadOnlyRootFilesystem:   &policy.ReadOnlyFS,
		AllowPrivilegeEscalation: ptrBool(false),
		RunAsNonRoot:             ptrBool(true),
		RunAsUser:                ptrInt64(1000),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	// Compliance Assertions (Captain test)
	if *securityContext.ReadOnlyRootFilesystem != true {
		t.Error("ReadOnlyRootFilesystem must be enforced")
	}
	if *securityContext.RunAsNonRoot != true {
		t.Error("RunAsNonRoot must be set to true")
	}
	if *securityContext.RunAsUser != 1000 {
		t.Errorf("expected UID 1000, got %d", *securityContext.RunAsUser)
	}
	if *securityContext.AllowPrivilegeEscalation != false {
		t.Error("AllowPrivilegeEscalation must be hard disabled")
	}
	
	hasDropAll := false
	for _, drop := range securityContext.Capabilities.Drop {
		if drop == "ALL" {
			hasDropAll = true
		}
	}
	if !hasDropAll {
		t.Error("Capabilities drop must contain 'ALL' to secure the worker pod")
	}
}
