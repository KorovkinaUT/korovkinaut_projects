package githubhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

const previewLimit = 200

// Client for GitHub updates requests
type Client struct {
	baseURL    string
	httpClient *http.Client
}

type UserResponse struct {
	Login string `json:"login"`
}

type IssueResponse struct {
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	CreatedAt   time.Time    `json:"created_at"`
	User        UserResponse `json:"user"`
	PullRequest *struct{}    `json:"pull_request,omitempty"`
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) GetRepositoryEvents(
	ctx context.Context,
	owner string,
	repo string,
) ([]update.GitHubEvent, error) {
	endpoint := fmt.Sprintf(
		"%s/repos/%s/%s/issues?state=all&sort=created&direction=desc",
		c.baseURL,
		owner,
		repo,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build github request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "link-tracker")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send github request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("github returned unexpected status: %s", resp.Status)
	}

	var issues []IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decode github response: %w", err)
	}

	events := make([]update.GitHubEvent, 0, len(issues))
	for _, issue := range issues {
		eventType := update.GitHubEventIssue
		if issue.PullRequest != nil {
			eventType = update.GitHubEventPullRequest
		}

		events = append(events, update.GitHubEvent{
			Type:         eventType,
			Title:        issue.Title,
			Username:     issue.User.Login,
			CreationTime: issue.CreatedAt,
			Preview:      buildPreview(issue.Body),
		})
	}

	return events, nil
}

func buildPreview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= previewLimit {
		return text
	}

	return text[:previewLimit]
}
