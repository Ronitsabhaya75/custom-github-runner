# 🚀 Sandboxed Kubernetes GitHub Actions Runner

[![ephemera-ci-pipeline](https://github.com/Ronitsabhaya75/custom-github-runner/actions/workflows/ci.yml/badge.svg)](https://github.com/Ronitsabhaya75/custom-github-runner/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Ronitsabhaya75/custom-github-runner)](https://github.com/Ronitsabhaya75/custom-github-runner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An ultra-lightweight, secure, and Kubernetes-native GitHub Actions runner built in Go. It dynamically resolves minimal, Alpine-based OCI runtimes for **32+ languages**, bypassing monolithic hosted virtual environments while enforcing rigorous zero-trust sandbox boundaries.

---

## 🏗️ Architecture & Component Design

```mermaid
flowchart TD
    subgraph GitHub
        GH_API[GitHub Actions Queue]
    end

    subgraph Go Orchestrator Daemon (runner)
        Parser[OCI Bundle Parser]
        Registry[Dynamic Language Registry]
        Engine[Pod Provisioning Loop]
    end

    subgraph Kubernetes Worker Node (Control Plane)
        K8s_API[Kubernetes API Server]
        
        subgraph Ephemeral Sandbox Pod
            Worker[Alpine Runner Container]
            NetPolicy[Egress Network Policy]
            Vol[Workspace emptyDir]
        end
    end

    GH_API <-->|Long-Poll| Go Orchestrator Daemon
    Parser -->|Parse config.json| Registry
    Registry -->|Resolve minimal image & cache| Engine
    Engine -->|client-go API| K8s_API
    K8s_API -->|Schedule Pod| Ephemeral Sandbox Pod
```

---

## ⚡ Key Features

* **Universal Language Registry:** Programmatically detects project dependencies and maps **32 different languages** to optimized Alpine-based container environments (e.g. Node, Go, Rust, Python, Java, etc.).
* **Security & Capability Hardening:** Translates OCI specifications (`config.json`) into Kubernetes `SecurityContext` definitions:
  * Forcefully enforces `ReadOnlyRootFilesystem: true`.
  * Locks run privileges to **non-root user UID 1000** (`RunAsNonRoot`).
  * Drops all default Linux capabilities (`CAP_SYS_ADMIN`, etc.) using `Capabilities.Drop: [ALL]`.
* **Zero-Trust Network Isolation:** Provisions a per-job `NetworkPolicy` that shuts down all inbound traffic and selectively blocks egress networks, sealing off local cloud metadata services (`169.254.169.254`) and private VPC zones.
* **Smart Dependency Caching:** Deterministically derives SHA256 cache keys from project dependency lock files (`go.sum`, `Cargo.lock`, `package-lock.json`) and mounts Kubernetes caching volumes automatically to accelerate rebuilds.
* **Dynamic Resource Tuning:** Automatically adjusts requested CPU cores and memory limits depending on the language profile (e.g. allocating more cores for Rust/Java compilers, and throttling micro-runtimes).
* **Dual Execution Engine:** Intelligently detects cluster connectivity. If the Kubernetes API is unreachable, it automatically pivots to a beautiful, high-fidelity **Simulation Mode** on your terminal to allow seamless offline workflows.

---

## 🛠️ Tech Stack & Dependencies

* **Go (Golang) 1.22+** — High-performance native daemon binary (~10MB footprint).
* **client-go** — Programmatic integration with Kubernetes API Server.
* **opencontainers/runtime-spec** — Native compliance with OCI container runtime specs.
* **GitHub API** — Polling engine for workflow job dispatch.

---

## 🚀 Getting Started

### Prerequisites

* Go 1.22 or higher installed.
* Kubernetes Cluster access (`kind`, `colima`, `minikube`, or cloud-native K8s) with Kubeconfig mapped to `~/.kube/config`.

### Compilation

Build the native runner executable using standard Go tools:

```bash
go build -o runner cmd/runner/main.go
```

### Running the Multi-Language CI Demo

Execute the compiled runner binary. It will auto-detect your cluster status, set up OCI boundaries, and run a **10-Language Pipeline Demo** showing side-by-side performance comparisons vs the official runner:

```bash
./runner
```

---

## 🧪 Comprehensive Testing Suite

We have implemented an industry-grade, multi-tier testing framework modeling the test specifications used by [**github.com/actions/runner-images**](https://github.com/actions/runner-images).

### 1. Offline Unit Tests
Verify dynamic language detection, resource recommendations, cache key calculations, and OCI configuration loaders offline:
```bash
go test -v ./pkg/sandbox/...
```

### 2. Live Capability Compliance & E2E Integration Tests
Run live verification pipelines that spin up test containers on your active K8s cluster, streaming stdout/stderr, and asserting that read-only filesystems, dropped capabilities, non-root users, and metadata egress blocks are perfectly enforced:
```bash
go test -tags=integration -v ./pkg/orchestrator/...
```

### 3. Automated CI Pipeline
All test suites are fully automated inside a matrix-based [**GitHub Actions Workflow**](.github/workflows/ci.yml) verifying compatibility across:
* 🐧 **Linux** (using real KinD integration clusters)
* 🍎 **macOS** (Smoke testing & Darwin compiler validations)
* 🏁 **Windows** (Smoke testing & MS-DOS/Powershell compiler validations)

---

## 🎯 Interview Talking Points (David's Blueprint)

If discussing this project with engineering leaders like **David** or Microsoft Upstream teams, focus on these low-level container engineering patterns:

1. **Pod Scheduling Under the Hood:** Explain how writing the Go `client-go` logic made you appreciate the asynchronous decoupling between the API Server, `etcd`, `kube-scheduler` (scoring nodes), and node `Kubelet` interactions.
2. **OCI Spec to Linux Kernel Translation:** Discuss how your Go loader reads `config.json` specs and translates them into kernel-level container rules: `ReadOnlyRootFilesystem` (`MS_RDONLY` mount flag), dropping capabilities (`capset` system call), and memory limits (kernel cgroups).
3. **Advanced Runtime Virtualization:** Highlight how your runner structure supports multiple Kubernetes `RuntimeClass` executors, allowing you to swap `runc` for user-space sandboxes like **gVisor (Sentry proxy)** or microVM hardware isolations like **Kata Containers** for untrusted code.
