package github

import (
	"context"
	"fmt"
	"net/http"

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

	if resp.StatusCode != http.StatusCreated {
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
	// List workflow runs currently in 'queued' or 'in_progress' states
	opts := &github.ListWorkflowRunsOptions{
		Status: "queued",
	}

	runs, resp, err := c.ghClient.Actions.ListRepositoryWorkflowRuns(ctx, c.owner, c.repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow runs: %w", err)
	}
	defer resp.Body.Close()

	var jobs []*RunnerJob
	for _, run := range runs.WorkflowRuns {
		// Simulating job extraction from active workflow runs
		jobs = append(jobs, &RunnerJob{
			ID:    run.GetID(),
			Name:  run.GetName(),
			Image: "alpine:latest", // Default fallback environment
			Steps: []string{
				"echo 'Initializing workspace'",
				"git clone " + run.GetRepository().GetHTMLURL(),
				"echo 'Workflow complete'",
			},
		})
	}

	return jobs, nil
}
