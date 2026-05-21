package sandbox

import (
	"testing"
)

func TestDetectLanguageFromCommands(t *testing.T) {
	tests := []struct {
		name     string
		commands []string
		expected Language
	}{
		{
			name:     "Python basic",
			commands: []string{"python3 --version", "pip install pytest"},
			expected: Python,
		},
		{
			name:     "Node basic",
			commands: []string{"node --version", "npm install express"},
			expected: Node,
		},
		{
			name:     "Go basic",
			commands: []string{"go version", "go build ./..."},
			expected: Go,
		},
		{
			name:     "Rust basic",
			commands: []string{"rustc --version", "cargo build --release"},
			expected: Rust,
		},
		{
			name:     "Java basic",
			commands: []string{"java --version", "mvn test"},
			expected: Java,
		},
		{
			name:     "Terraform infrastructure",
			commands: []string{"terraform init", "terraform plan"},
			expected: Terraform,
		},
		{
			name:     "Shell scripts",
			commands: []string{"#!/bin/bash", "echo 'running bash script'"},
			expected: Shell,
		},
		{
			name:     "Generic fallback",
			commands: []string{"echo 'standard echo command'", "whoami"},
			expected: Lightweight,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := DetectLanguageFromCommands(tt.commands)
			if actual != tt.expected {
				t.Errorf("expected language %s, got %s", tt.expected, actual)
			}
		})
	}
}

func TestResolveRuntimeImage(t *testing.T) {
	tests := []struct {
		name     string
		language Language
		expected string
	}{
		{"Go resolved", Go, "golang:1.22-alpine"},
		{"Python resolved", Python, "python:3.12-alpine"},
		{"Node resolved", Node, "node:20-alpine"},
		{"Rust resolved", Rust, "rust:1.77-alpine"},
		{"PHP resolved", PHP, "php:8.3-alpine"},
		{"Generic fallback", Lightweight, "alpine:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ResolveRuntimeImage(tt.language)
			if actual != tt.expected {
				t.Errorf("expected image %s, got %s", tt.expected, actual)
			}
		})
	}
}
