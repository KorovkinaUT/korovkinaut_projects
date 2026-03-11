package githubhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client for github updates requests
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// For github response
type RepositoryResponse struct {
	UpdatedAt time.Time `json:"updated_at"`
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) GetRepositoryUpdatedAt(
	ctx context.Context,
	owner string,
	repo string,
) (time.Time, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("build github request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "link-tracker")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("send github request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return time.Time{}, fmt.Errorf("github returned unexpected status: %s", resp.Status)
	}

	var repository RepositoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return time.Time{}, fmt.Errorf("decode github response: %w", err)
	}

	return repository.UpdatedAt, nil
}