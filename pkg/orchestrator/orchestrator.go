package orchestrator

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ronitsabhaya/k8s-github-runner/pkg/sandbox"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Job represents a single CI workflow job to run
type Job struct {
	ID          string
	Image       string
	Commands    []string
	Namespace   string
	RuntimeClass string // e.g. "runc", "gvisor", "kata"
}

// RunnerOrchestrator manages the lifecycle of ephemeral CI pods
type RunnerOrchestrator struct {
	clientset *kubernetes.Clientset
	DryRun    bool
}

// NewRunnerOrchestrator initializes the K8s clientset using kubeconfig or in-cluster config
func NewRunnerOrchestrator(kubeconfigPath string) (*RunnerOrchestrator, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Printf("⚠️  [Orchestrator] Kubeconfig could not be parsed. Switching to High-Fidelity Sandbox Simulation Mode.\n")
		return &RunnerOrchestrator{DryRun: true}, nil
	}

	// Restrict connection timeout to 2 seconds to avoid hanging under heavy disk throttling
	config.Timeout = 2 * time.Second
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("⚠️  [Orchestrator] Failed to instantiate K8s client. Switching to High-Fidelity Sandbox Simulation Mode.\n")
		return &RunnerOrchestrator{DryRun: true}, nil
	}

	// Verify live connection to the cluster API endpoint (RBAC-safe discovery check)
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		fmt.Printf("⚠️  [Orchestrator] Local cluster endpoint unreachable. Switching to High-Fidelity Sandbox Simulation Mode to prevent disk I/O freezes.\n")
		return &RunnerOrchestrator{DryRun: true}, nil
	}

	return &RunnerOrchestrator{clientset: clientset, DryRun: false}, nil
}

// ScheduleJob launches an ephemeral pod, enforces policy boundaries, and monitors execution
func (o *RunnerOrchestrator) ScheduleJob(ctx context.Context, job Job, policy *sandbox.Policy) error {
	if o.DryRun {
		return o.executeSimulatedJob(ctx, job, policy)
	}

	podName := fmt.Sprintf("runner-job-%s", job.ID)
	fmt.Printf("[Orchestrator] Preparing sandbox pod: %s\n", podName)

	// Translate Sandbox Policy into K8s Container Security Context
	securityContext := &corev1.SecurityContext{
		ReadOnlyRootFilesystem:   &policy.ReadOnlyFS,
		AllowPrivilegeEscalation: ptrBool(false),
		RunAsNonRoot:             ptrBool(true),
		RunAsUser:                ptrInt64(1000), // Default non-root user
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	// Dynamic Resource Limits
	resources := corev1.ResourceRequirements{
		Limits:   make(corev1.ResourceList),
		Requests: make(corev1.ResourceList),
	}
	if policy.MaxMemoryMB > 0 {
		memLimit := resource.NewQuantity(policy.MaxMemoryMB*1024*1024, resource.BinarySI)
		resources.Limits[corev1.ResourceMemory] = *memLimit
		resources.Requests[corev1.ResourceMemory] = *memLimit
	}

	// Command execution layout
	// Standard entrypoint for runner container to execute CI steps
	var command []string
	var args []string
	if len(job.Commands) > 0 {
		command = []string{"/bin/sh", "-c"}
		// Join commands into a single script execution
		script := ""
		for _, cmd := range job.Commands {
			script += fmt.Sprintf("echo '==> Running: %s'\n%s\n", cmd, cmd)
		}
		args = []string{script}
	}

	// Define the Ephemeral Runner Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"app":  "custom-github-runner",
				"role": "worker",
				"job":  job.ID,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:            "runner-executor",
					Image:           job.Image,
					Command:         command,
					Args:            args,
					SecurityContext: securityContext,
					Resources:       resources,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace",
							MountPath: "/workspace",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	// Assign the Runtime Class (runc, gvisor, kata) if specified
	if job.RuntimeClass != "" {
		pod.Spec.RuntimeClassName = &job.RuntimeClass
		fmt.Printf("[Orchestrator] Applying RuntimeClass: %s (OCI Sandbox)\n", job.RuntimeClass)
	}

	// Spawn the Pod in Kubernetes
	podsClient := o.clientset.CoreV1().Pods(job.Namespace)
	_, err := podsClient.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}
	fmt.Printf("[Orchestrator] Pod %s successfully scheduled in namespace %s\n", podName, job.Namespace)

	// Deploy strict NetworkPolicy security boundaries
	err = o.EnsureNetworkPolicy(ctx, job.Namespace, podName, policy.AllowNetwork)
	if err != nil {
		fmt.Printf("[Orchestrator] Warning: Failed to enforce NetworkPolicy: %v\n", err)
	}

	// Ensure cleanup on return
	defer func() {
		fmt.Printf("[Orchestrator] Cleaning up pod: %s...\n", podName)
		
		// Clean up network policy
		_ = o.CleanNetworkPolicy(context.Background(), job.Namespace, podName)

		deleteGracePeriod := int64(0)
		err := podsClient.Delete(context.Background(), podName, metav1.DeleteOptions{
			GracePeriodSeconds: &deleteGracePeriod,
		})
		if err != nil {
			fmt.Printf("[Orchestrator] Warning: Failed to clean up pod %s: %v\n", podName, err)
		} else {
			fmt.Printf("[Orchestrator] Pod %s successfully deleted.\n", podName)
		}
	}()

	// Wait for the Pod to start and stream logs
	return o.monitorAndStreamLogs(ctx, job.Namespace, podName)
}

func (o *RunnerOrchestrator) monitorAndStreamLogs(ctx context.Context, namespace, podName string) error {
	podsClient := o.clientset.CoreV1().Pods(namespace)

	// Wait for Pod to become active / running
	for {
		pod, err := podsClient.Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error waiting for pod state: %w", err)
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
			fmt.Println("[Orchestrator] Pod is pending, waiting for schedule/image pull...")
			time.Sleep(2 * time.Second)
		case corev1.PodRunning, corev1.PodSucceeded:
			fmt.Println("[Orchestrator] Pod is running, starting log stream...")
			return o.streamLogs(ctx, namespace, podName)
		case corev1.PodFailed:
			return fmt.Errorf("pod failed to start or container crashed immediately")
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (o *RunnerOrchestrator) streamLogs(ctx context.Context, namespace, podName string) error {
	req := o.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Follow: true,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to open log stream: %w", err)
	}
	defer stream.Close()

	// Print logs in real-time to stdout
	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			fmt.Print(string(buf[:n]))
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading log stream: %w", err)
		}
	}

	return nil
}

func ptrBool(b bool) *bool {
	return &b
}

func ptrInt64(i int64) *int64 {
	return &i
}
