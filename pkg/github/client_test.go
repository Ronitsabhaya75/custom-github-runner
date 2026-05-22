package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorkflowYAML(t *testing.T) {
	yamlContent := []byte(`
name: custom-ci-pipeline
on: [push]
jobs:
  test-job:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
      - name: Setup Go environment
        uses: actions/setup-go@v5
        with:
          go-version: '1.22.x'
      - name: Run unit test steps
        run: go test -v ./...
`)

	wf, err := ParseWorkflowYAML(yamlContent)
	if err != nil {
		t.Fatalf("ParseWorkflowYAML failed: %v", err)
	}

	if wf.Name != "custom-ci-pipeline" {
		t.Errorf("expected workflow name 'custom-ci-pipeline', got '%s'", wf.Name)
	}

	job, ok := wf.Jobs["test-job"]
	if !ok {
		t.Fatal("expected 'test-job' to be defined in jobs list")
	}

	if job.RunsOn != "ubuntu-latest" {
		t.Errorf("expected runs-on to be 'ubuntu-latest', got '%s'", job.RunsOn)
	}

	if len(job.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(job.Steps))
	}

	step1 := job.Steps[0]
	if step1.Name != "Checkout repository" {
		t.Errorf("unexpected step name: %s", step1.Name)
	}
	if step1.Uses != "actions/checkout@v4" {
		t.Errorf("unexpected step uses: %s", step1.Uses)
	}

	step3 := job.Steps[2]
	if step3.Name != "Run unit test steps" {
		t.Errorf("unexpected step name: %s", step3.Name)
	}
	if step3.Run != "go test -v ./..." {
		t.Errorf("unexpected step run command: %s", step3.Run)
	}
}

func TestRunnerConfigSaveLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runner-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configFilePath := filepath.Join(tempDir, ".runner-config.json")

	cfg := &RunnerConfig{
		Token:    "ghp_testpersonalaccesstoken",
		Owner:    "test-user",
		Repo:     "test-repo",
		Name:     "runner-unit-test",
		IsConfig: true,
	}

	// Save settings
	if err := SaveConfig(configFilePath, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load settings
	loaded, err := LoadConfig(configFilePath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Token != cfg.Token {
		t.Errorf("expected Token '%s', got '%s'", cfg.Token, loaded.Token)
	}
	if loaded.Owner != cfg.Owner {
		t.Errorf("expected Owner '%s', got '%s'", cfg.Owner, loaded.Owner)
	}
	if loaded.Repo != cfg.Repo {
		t.Errorf("expected Repo '%s', got '%s'", cfg.Repo, loaded.Repo)
	}
	if loaded.Name != cfg.Name {
		t.Errorf("expected Name '%s', got '%s'", cfg.Name, loaded.Name)
	}
	if !loaded.IsConfig {
		t.Error("expected IsConfig to be true")
	}
}
