package github

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Step represents a single sequential action within a job step
type Step struct {
	Name string            `yaml:"name,omitempty"`
	Run  string            `yaml:"run,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
}

// JobDefinition describes a containerized job context inside the workflow YAML
type JobDefinition struct {
	Name   string   `yaml:"name,omitempty"`
	RunsOn string   `yaml:"runs-on,omitempty"`
	Steps  []*Step  `yaml:"steps"`
}

// Workflow models the root object of a GitHub Actions workflow yaml
type Workflow struct {
	Name string                    `yaml:"name,omitempty"`
	Jobs map[string]*JobDefinition `yaml:"jobs"`
}

// ParseWorkflowYAML decodes standard YAML syntax into a high-level Workflow definition
func ParseWorkflowYAML(content []byte) (*Workflow, error) {
	var wf Workflow
	if err := yaml.Unmarshal(content, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}
	return &wf, nil
}
