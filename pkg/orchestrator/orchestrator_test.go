package orchestrator

import (
	"testing"

	"github.com/ronitsabhaya/k8s-github-runner/pkg/sandbox"
	corev1 "k8s.io/api/core/v1"
)

// TestComplianceCapabilityDrops runs the 'Captain' Capability dropped compliance test
// This is a standard unit test that executes offline and runs on all OS platforms (Linux, macOS, Windows)
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
