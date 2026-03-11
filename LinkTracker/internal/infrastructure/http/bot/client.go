package bothttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client for processing bot server responces
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) SendUpdate(update LinkUpdate) error {
	endpoint := fmt.Sprintf("%s/updates", c.baseURL)

	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal send update request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build send update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return parseAPIError(resp)
}

func parseAPIError(resp *http.Response) error {
	var apiErr ApiErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("unexpected status %d and failed to decode error response: %w", resp.StatusCode, err)
	}

	return fmt.Errorf(
		"bot api error: status=%d code=%s description=%s exception=%s message=%s",
		resp.StatusCode,
		apiErr.Code,
		apiErr.Description,
		apiErr.ExceptionName,
		apiErr.ExceptionMessage,
	)
}
