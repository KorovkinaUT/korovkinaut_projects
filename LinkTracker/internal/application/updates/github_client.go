package updates

import (
	"context"
	"fmt"
	"time"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
)

type GitHubClient struct {
	client *githubhttp.Client
}

func NewGitHubClient(client *githubhttp.Client) *GitHubClient {
	return &GitHubClient{
		client: client,
	}
}

func (c *GitHubClient) Type() schedulerlink.LinkType {
	return schedulerlink.TypeGitHub
}

func (c *GitHubClient) GetUpdatedAt(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
	githubLink, ok := link.(schedulerlink.GitHubLink)
	if !ok {
		return time.Time{}, fmt.Errorf("expected github link, got %T", link)
	}

	updatedAt, err := c.client.GetRepositoryUpdatedAt(ctx, githubLink.Owner, githubLink.Repo)
	if err != nil {
		return time.Time{}, fmt.Errorf("get github updated at: %w", err)
	}

	return updatedAt, nil
}