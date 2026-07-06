package scrapperhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/avast/retry-go"
	"github.com/sony/gobreaker"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	httpinfra "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http"
)

// Client for processing scrapper server responses
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
		circuitBreaker: httpinfra.NewCircuitBreaker("scrapper-http-client", cbCfg),
	}
}

func (c *Client) RegisterChat(ctx context.Context, chatID int64) error {
	_, err := c.circuitBreaker.Execute(func() (any, error) {
		return nil, retry.Do(
			func() error {
				return c.registerChat(ctx, chatID)
			},
			retry.Attempts(c.retryCfg.Attempts),
			retry.Delay(c.retryCfg.Delay),
			retry.DelayType(retry.FixedDelay),
			retry.Context(ctx),
		)
	})

	return err
}

func (c *Client) registerChat(ctx context.Context, chatID int64) error {
	endpoint := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return retry.Unrecoverable(fmt.Errorf("build register chat request: %w", err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return retry.Unrecoverable(fmt.Errorf("send register chat request: %w", err))
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return retry.Unrecoverable(repository.ErrChatAlreadyExists)
	default:
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return fmt.Errorf("scrapper api returned retryable status: %s", resp.Status)
		}

		return retry.Unrecoverable(parseAPIError(resp))
	}
}

func (c *Client) DeleteChat(ctx context.Context, chatID int64) error {
	_, err := c.circuitBreaker.Execute(func() (any, error) {
		return nil, retry.Do(
			func() error {
				return c.deleteChat(ctx, chatID)
			},
			retry.Attempts(c.retryCfg.Attempts),
			retry.Delay(c.retryCfg.Delay),
			retry.DelayType(retry.FixedDelay),
			retry.Context(ctx),
		)
	})

	return err
}

func (c *Client) deleteChat(ctx context.Context, chatID int64) error {
	endpoint := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return retry.Unrecoverable(fmt.Errorf("build delete chat request: %w", err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return retry.Unrecoverable(fmt.Errorf("send delete chat request: %w", err))
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return retry.Unrecoverable(repository.ErrChatNotFound)
	default:
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return fmt.Errorf("scrapper api returned retryable status: %s", resp.Status)
		}

		return retry.Unrecoverable(parseAPIError(resp))
	}
}

func (c *Client) ListLinks(ctx context.Context, chatID int64) (ListLinksResponse, error) {
	var result ListLinksResponse

	_, err := c.circuitBreaker.Execute(func() (any, error) {
		err := retry.Do(
			func() error {
				response, err := c.listLinks(ctx, chatID)
				if err != nil {
					return err
				}

				result = response

				return nil
			},
			retry.Attempts(c.retryCfg.Attempts),
			retry.Delay(c.retryCfg.Delay),
			retry.DelayType(retry.FixedDelay),
			retry.Context(ctx),
		)

		return nil, err
	})
	if err != nil {
		return ListLinksResponse{}, err
	}

	return result, nil
}

func (c *Client) listLinks(ctx context.Context, chatID int64) (ListLinksResponse, error) {
	endpoint := fmt.Sprintf("%s/links", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ListLinksResponse{}, retry.Unrecoverable(fmt.Errorf("build list links request: %w", err))
	}

	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ListLinksResponse{}, retry.Unrecoverable(fmt.Errorf("send list links request: %w", err))
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result ListLinksResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return ListLinksResponse{}, retry.Unrecoverable(fmt.Errorf("decode list links response: %w", err))
		}

		return result, nil
	case http.StatusNotFound:
		return ListLinksResponse{}, retry.Unrecoverable(repository.ErrChatNotFound)
	default:
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return ListLinksResponse{}, fmt.Errorf("scrapper api returned retryable status: %s", resp.Status)
		}

		return ListLinksResponse{}, retry.Unrecoverable(parseAPIError(resp))
	}
}

func (c *Client) AddLink(ctx context.Context, chatID int64, request AddLinkRequest) (LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return LinkResponse{}, fmt.Errorf("marshal add link request: %w", err)
	}

	var result LinkResponse

	_, err = c.circuitBreaker.Execute(func() (any, error) {
		err := retry.Do(
			func() error {
				response, err := c.addLink(ctx, chatID, body)
				if err != nil {
					return err
				}

				result = response

				return nil
			},
			retry.Attempts(c.retryCfg.Attempts),
			retry.Delay(c.retryCfg.Delay),
			retry.DelayType(retry.FixedDelay),
			retry.Context(ctx),
		)

		return nil, err
	})
	if err != nil {
		return LinkResponse{}, err
	}

	return result, nil
}

func (c *Client) addLink(ctx context.Context, chatID int64, body []byte) (LinkResponse, error) {
	endpoint := fmt.Sprintf("%s/links", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return LinkResponse{}, retry.Unrecoverable(fmt.Errorf("build add link request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return LinkResponse{}, retry.Unrecoverable(fmt.Errorf("send add link request: %w", err))
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result LinkResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return LinkResponse{}, retry.Unrecoverable(fmt.Errorf("decode add link response: %w", err))
		}

		return result, nil
	case http.StatusNotFound:
		return LinkResponse{}, retry.Unrecoverable(repository.ErrChatNotFound)
	case http.StatusConflict:
		return LinkResponse{}, retry.Unrecoverable(repository.ErrLinkAlreadyTracked)
	default:
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return LinkResponse{}, fmt.Errorf("scrapper api returned retryable status: %s", resp.Status)
		}

		return LinkResponse{}, retry.Unrecoverable(parseAPIError(resp))
	}
}

func (c *Client) RemoveLink(ctx context.Context, chatID int64, request RemoveLinkRequest) (LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return LinkResponse{}, fmt.Errorf("marshal remove link request: %w", err)
	}

	var result LinkResponse

	_, err = c.circuitBreaker.Execute(func() (any, error) {
		err := retry.Do(
			func() error {
				response, err := c.removeLink(ctx, chatID, body)
				if err != nil {
					return err
				}

				result = response

				return nil
			},
			retry.Attempts(c.retryCfg.Attempts),
			retry.Delay(c.retryCfg.Delay),
			retry.DelayType(retry.FixedDelay),
			retry.Context(ctx),
		)

		return nil, err
	})
	if err != nil {
		return LinkResponse{}, err
	}

	return result, nil
}

func (c *Client) removeLink(ctx context.Context, chatID int64, body []byte) (LinkResponse, error) {
	endpoint := fmt.Sprintf("%s/links", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, bytes.NewReader(body))
	if err != nil {
		return LinkResponse{}, retry.Unrecoverable(fmt.Errorf("build remove link request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return LinkResponse{}, retry.Unrecoverable(fmt.Errorf("send remove link request: %w", err))
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result LinkResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return LinkResponse{}, retry.Unrecoverable(fmt.Errorf("decode remove link response: %w", err))
		}

		return result, nil
	case http.StatusNotFound:
		return LinkResponse{}, retry.Unrecoverable(repository.ErrChatNotFound)
	default:
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return LinkResponse{}, fmt.Errorf("scrapper api returned retryable status: %s", resp.Status)
		}

		return LinkResponse{}, retry.Unrecoverable(parseAPIError(resp))
	}
}

func parseAPIError(resp *http.Response) error {
	var apiErr ApiErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("unexpected status %d and failed to decode error response: %w", resp.StatusCode, err)
	}

	return fmt.Errorf(
		"scrapper api error: status=%d code=%s description=%s exception=%s message=%s",
		resp.StatusCode,
		apiErr.Code,
		apiErr.Description,
		apiErr.ExceptionName,
		apiErr.ExceptionMessage,
	)
}
