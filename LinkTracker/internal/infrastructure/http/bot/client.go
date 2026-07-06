package bothttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avast/retry-go"
	"github.com/sony/gobreaker"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	httpinfra "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http"
)

type Client struct {
	baseURL        string
	httpClient     *http.Client
	retryCfg       *config.RetryConfig
	circuitBreaker *gobreaker.CircuitBreaker
}

func NewClient(
	baseURL string,
	httpClient *http.Client,
	retryCfg *config.RetryConfig,
	cbCfg *config.CircuitBreakerConfig,
) *Client {
	return &Client{
		baseURL:        baseURL,
		httpClient:     httpClient,
		retryCfg:       retryCfg,
		circuitBreaker: httpinfra.NewCircuitBreaker("bot-http-client", cbCfg),
	}
}

func (c *Client) SendUpdate(ctx context.Context, update LinkUpdate) error {
	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal send update request: %w", err)
	}

	_, err = c.circuitBreaker.Execute(func() (any, error) {
		return nil, retry.Do(
			func() error {
				return c.sendUpdate(ctx, body)
			},
			retry.Attempts(c.retryCfg.Attempts),
			retry.Delay(c.retryCfg.Delay),
			retry.DelayType(retry.FixedDelay),
			retry.Context(ctx),
		)
	})

	return err
}

func (c *Client) sendUpdate(ctx context.Context, body []byte) error {
	endpoint := fmt.Sprintf("%s/updates", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return retry.Unrecoverable(fmt.Errorf("build send update request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return retry.Unrecoverable(fmt.Errorf("send update request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
		return fmt.Errorf("retryable bot api status: %d", resp.StatusCode)
	}

	return retry.Unrecoverable(parseAPIError(resp))
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
