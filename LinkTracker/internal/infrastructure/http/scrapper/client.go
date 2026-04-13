package scrapperhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
)

// Client for processing scrapper server respones
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

func (c *Client) RegisterChat(ctx context.Context, chatID int64) error {
	endpoint := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build register chat request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send register chat request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return repository.ErrChatAlreadyExists
	default:
		return parseAPIError(resp)
	}
}

func (c *Client) DeleteChat(ctx context.Context, chatID int64) error {
	endpoint := fmt.Sprintf("%s/tg-chat/%d", c.baseURL, chatID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build delete chat request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send delete chat request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return repository.ErrChatNotFound
	default:
		return parseAPIError(resp)
	}
}

func (c *Client) ListLinks(ctx context.Context, chatID int64) (ListLinksResponse, error) {
	endpoint := fmt.Sprintf("%s/links", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ListLinksResponse{}, fmt.Errorf("build list links request: %w", err)
	}

	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ListLinksResponse{}, fmt.Errorf("send list links request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result ListLinksResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return ListLinksResponse{}, fmt.Errorf("decode list links response: %w", err)
		}
		return result, nil
	case http.StatusNotFound:
		return ListLinksResponse{}, repository.ErrChatNotFound
	default:
		return ListLinksResponse{}, parseAPIError(resp)
	}
}

func (c *Client) AddLink(ctx context.Context, chatID int64, request AddLinkRequest) (LinkResponse, error) {
	endpoint := fmt.Sprintf("%s/links", c.baseURL)

	body, err := json.Marshal(request)
	if err != nil {
		return LinkResponse{}, fmt.Errorf("marshal add link request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return LinkResponse{}, fmt.Errorf("build add link request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return LinkResponse{}, fmt.Errorf("send add link request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result LinkResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return LinkResponse{}, fmt.Errorf("decode add link response: %w", err)
		}
		return result, nil
	case http.StatusNotFound:
		return LinkResponse{}, repository.ErrChatNotFound
	case http.StatusConflict:
		return LinkResponse{}, repository.ErrLinkAlreadyTracked
	default:
		return LinkResponse{}, parseAPIError(resp)
	}
}

func (c *Client) RemoveLink(ctx context.Context, chatID int64, request RemoveLinkRequest) (LinkResponse, error) {
	endpoint := fmt.Sprintf("%s/links", c.baseURL)

	body, err := json.Marshal(request)
	if err != nil {
		return LinkResponse{}, fmt.Errorf("marshal remove link request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, bytes.NewReader(body))
	if err != nil {
		return LinkResponse{}, fmt.Errorf("build remove link request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Tg-Chat-Id", strconv.FormatInt(chatID, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return LinkResponse{}, fmt.Errorf("send remove link request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result LinkResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return LinkResponse{}, fmt.Errorf("decode remove link response: %w", err)
		}
		return result, nil
	case http.StatusNotFound:
		return LinkResponse{}, repository.ErrChatNotFound
	default:
		return LinkResponse{}, parseAPIError(resp)
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
