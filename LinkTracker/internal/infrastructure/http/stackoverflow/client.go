package stackoverflowhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client for StackOverflow updates requests
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// For StackOverflow response
type QuestionResponse struct {
	Items []Question `json:"items"`
}

type Question struct {
	LastActivityDate int64 `json:"last_activity_date"`
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) GetQuestionUpdatedAt(
	ctx context.Context,
	questionID string,
) (time.Time, error) {
	endpoint := fmt.Sprintf("%s/questions/%s?site=stackoverflow", c.baseURL, questionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("build stackoverflow request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("send stackoverflow request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return time.Time{}, fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status)
	}

	var questionResponse QuestionResponse
	if err := json.NewDecoder(resp.Body).Decode(&questionResponse); err != nil {
		return time.Time{}, fmt.Errorf("decode stackoverflow response: %w", err)
	}

	if len(questionResponse.Items) == 0 {
		return time.Time{}, fmt.Errorf("stackoverflow question not found")
	}

	return time.Unix(questionResponse.Items[0].LastActivityDate, 0), nil
}
