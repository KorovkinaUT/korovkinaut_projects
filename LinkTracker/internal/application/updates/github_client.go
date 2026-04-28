package updates

import (
	"context"
	"fmt"
	"time"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
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

func (c *GitHubClient) GetNewEvents(
	ctx context.Context,
	link schedulerlink.SchedulerLink,
	since time.Time,
) ([]update.Event, error) {
	githubLink, ok := link.(schedulerlink.GitHubLink)
	if !ok {
		return nil, fmt.Errorf("expected github link, got %T", link)
	}

	events, err := c.client.GetRepositoryEvents(ctx, githubLink.Owner, githubLink.Repo, since)
	if err != nil {
		return nil, fmt.Errorf("get github events: %w", err)
	}

	result := make([]update.Event, 0, len(events))
	for _, event := range events {
		result = append(result, event)
	}

	return result, nil
}
