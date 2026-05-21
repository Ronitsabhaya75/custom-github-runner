package orchestrator

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EnsureNetworkPolicy provisions a strict network isolation policy for the CI executor pod
func (o *RunnerOrchestrator) EnsureNetworkPolicy(ctx context.Context, namespace, podName string, allowNetwork bool) error {
	policyName := fmt.Sprintf("isolate-%s", podName)
	fmt.Printf("[Security] Configuring NetworkPolicy: %s (AllowNetwork=%t)\n", policyName, allowNetwork)

	// Define ingress/egress rules based on OCI sandbox settings
	var egressRules []networkingv1.NetworkPolicyEgressRule

	if allowNetwork {
		// Allow external internet egress but explicitly block local network metadata addresses (e.g., AWS/GCP metadata endpoints)
		egressRules = []networkingv1.NetworkPolicyEgressRule{
			{
				To: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: &networkingv1.IPBlock{
							CIDR: "0.0.0.0/0",
							Except: []string{
								"169.254.169.254/32", // Cloud Metadata Server
								"10.0.0.0/8",         // Private LAN / VPC networks
								"172.16.0.0/12",
								"192.168.0.0/16",
							},
						},
					},
				},
			},
		}
	} else {
		// Deny all network traffic (completely isolated offline sandbox)
		egressRules = []networkingv1.NetworkPolicyEgressRule{}
	}

	netPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: namespace,
			Labels: map[string]string{
				"associated-job-pod": podName,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  "custom-github-runner",
					"role": "worker",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			// Block all incoming connections to the running CI workspace container
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
			Egress:  egressRules,
		},
	}

	// Deploy NetworkPolicy to cluster
	netClient := o.clientset.NetworkingV1().NetworkPolicies(namespace)
	_, err := netClient.Create(ctx, netPolicy, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create network policy: %w", err)
	}

	fmt.Printf("[Security] NetworkPolicy %s successfully deployed.\n", policyName)
	return nil
}

// CleanNetworkPolicy removes the associated network isolation policy
func (o *RunnerOrchestrator) CleanNetworkPolicy(ctx context.Context, namespace, podName string) error {
	policyName := fmt.Sprintf("isolate-%s", podName)
	netClient := o.clientset.NetworkingV1().NetworkPolicies(namespace)
	
	gracePeriod := int64(0)
	err := netClient.Delete(ctx, policyName, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})
	if err != nil {
		return fmt.Errorf("failed to delete network policy: %w", err)
	}

	fmt.Printf("[Security] NetworkPolicy %s cleaned up.\n", policyName)
	return nil
}

// Helper to construct Port types if needed for DNS resolution mapping
func ptrIntStr(val int) *intstr.IntOrString {
	res := intstr.FromInt32(int32(val))
	return &res
}
