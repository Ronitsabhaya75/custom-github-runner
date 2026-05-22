package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Client handles communications with the GitHub Actions API
type Client struct {
	ghClient *github.Client
	owner    string
	repo     string
}

// NewClient initializes a new GitHub API client using a Personal Access Token (PAT)
func NewClient(ctx context.Context, token, owner, repo string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	return &Client{
		ghClient: ghClient,
		owner:    owner,
		repo:     repo,
	}
}

// GetRegistrationToken requests a runner registration token from GitHub Actions
func (c *Client) GetRegistrationToken(ctx context.Context) (string, error) {
	fmt.Printf("[GitHub] Fetching runner registration token for %s/%s...\n", c.owner, c.repo)
	
	token, resp, err := c.ghClient.Actions.CreateRegistrationToken(ctx, c.owner, c.repo)
	if err != nil {
		return "", fmt.Errorf("failed to create registration token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to retrieve token: status %d", resp.StatusCode)
	}

	return token.GetToken(), nil
}

// RunnerJob represents a simplified workflow job extracted from GitHub APIs
type RunnerJob struct {
	ID       int64
	Name     string
	Steps    []string
	Image    string
}

// FetchPendingJobs checks the repository actions queues for queued runner tasks
func (c *Client) FetchPendingJobs(ctx context.Context) ([]*RunnerJob, error) {
	// List workflow runs currently in 'queued' states
	opts := &github.ListWorkflowRunsOptions{
		Status: "queued",
	}

	runs, resp, err := c.ghClient.Actions.ListRepositoryWorkflowRuns(ctx, c.owner, c.repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow runs: %w", err)
	}
	defer resp.Body.Close()

	// Proactively pull workflow definitions from .github/workflows in the repository
	workflows, err := c.FetchRepositoryWorkflows(ctx)
	if err != nil {
		fmt.Printf("[GitHub] Warning: failed to fetch real workflows from repository: %v. Using mock steps.\n", err)
	}

	var jobs []*RunnerJob
	for _, run := range runs.WorkflowRuns {
		// Look up parsed steps from loaded workflows matching this run
		runName := run.GetName()
		var steps []string

		if len(workflows) > 0 {
			for _, wf := range workflows {
				if wf.Name == runName || strings.Contains(strings.ToLower(runName), strings.ToLower(wf.Name)) {
					// Extract steps from the first job defined
					for _, jobDef := range wf.Jobs {
						for _, step := range jobDef.Steps {
							if step.Run != "" {
								steps = append(steps, step.Run)
							} else if step.Uses != "" {
								steps = append(steps, fmt.Sprintf("echo 'Executing action: %s'", step.Uses))
							}
						}
					}
				}
			}
		}

		// Fallback steps if no workflows match
		if len(steps) == 0 {
			steps = []string{
				"echo 'Initializing workspace'",
				"git clone " + run.GetRepository().GetHTMLURL(),
				"echo 'Workflow complete'",
			}
		}

		jobs = append(jobs, &RunnerJob{
			ID:    run.GetID(),
			Name:  runName,
			Image: "alpine:latest", // Default fallback environment
			Steps: steps,
		})
	}

	return jobs, nil
}

// FetchRepositoryWorkflows reads all files from .github/workflows/ and parses them
func (c *Client) FetchRepositoryWorkflows(ctx context.Context) ([]*Workflow, error) {
	_, directoryContent, resp, err := c.ghClient.Repositories.GetContents(ctx, c.owner, c.repo, ".github/workflows", &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf(".github/workflows folder not found in repository")
		}
		return nil, err
	}

	var workflows []*Workflow
	for _, item := range directoryContent {
		name := item.GetName()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}

		fileContent, _, _, err := c.ghClient.Repositories.GetContents(ctx, c.owner, c.repo, item.GetPath(), &github.RepositoryContentGetOptions{})
		if err != nil {
			continue
		}

		rawContent, err := fileContent.GetContent()
		if err != nil {
			// Fallback: If content is empty or needs base64 decode from Content field
			if fileContent.Content != nil {
				decoded, err := base64.StdEncoding.DecodeString(*fileContent.Content)
				if err == nil {
					rawContent = string(decoded)
				}
			}
		}

		if rawContent == "" {
			continue
		}

		wf, err := ParseWorkflowYAML([]byte(rawContent))
		if err != nil {
			fmt.Printf("[GitHub] Warning: failed to parse workflow %s: %v\n", name, err)
			continue
		}

		// Use filename as workflow name if empty
		if wf.Name == "" {
			wf.Name = strings.TrimSuffix(strings.TrimSuffix(name, ".yml"), ".yaml")
		}

		workflows = append(workflows, wf)
	}

	return workflows, nil
}
