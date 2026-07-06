package githubhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/sony/gobreaker"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	httpinfra "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http"
)

const previewLimit = 200

// Client for GitHub updates requests
type Client struct {
	baseURL        string
	httpClient     *http.Client
	retryCfg       *config.RetryConfig
	circuitBreaker *gobreaker.CircuitBreaker
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

type issuesPageResponse struct {
	issues     []IssueResponse
	linkHeader string
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
		circuitBreaker: httpinfra.NewCircuitBreaker("github-http-client", cbCfg),
	}
}

func (c *Client) GetRepositoryEvents(
	ctx context.Context,
	owner string,
	repo string,
	since time.Time,
) ([]update.GitHubEvent, error) {
	params := url.Values{}
	params.Set("state", "all")
	params.Set("sort", "created")
	params.Set("direction", "desc")
	params.Set("since", since.Format(time.RFC3339))
	params.Set("per_page", "100")

	events := make([]update.GitHubEvent, 0)
	page := 1

	for {
		params.Set("page", strconv.Itoa(page))

		endpoint := fmt.Sprintf(
			"%s/repos/%s/%s/issues?%s",
			c.baseURL,
			owner,
			repo,
			params.Encode(),
		)

		pageResponse, err := c.getIssuesPage(ctx, endpoint)
		if err != nil {
			return nil, err
		}

		for _, issue := range pageResponse.issues {
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

		// checks if there is next page
		if !strings.Contains(pageResponse.linkHeader, `rel="next"`) {
			break
		}

		page++
	}

	return events, nil
}

func (c *Client) getIssuesPage(ctx context.Context, endpoint string) (*issuesPageResponse, error) {
	var result *issuesPageResponse

	_, err := c.circuitBreaker.Execute(func() (any, error) {
		err := retry.Do(
			func() error {
				response, err := c.sendIssuesPageRequest(ctx, endpoint)
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
		return nil, err
	}

	return result, nil
}

func (c *Client) sendIssuesPageRequest(ctx context.Context, endpoint string) (*issuesPageResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, retry.Unrecoverable(fmt.Errorf("build github request: %w", err))
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "link-tracker")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, retry.Unrecoverable(fmt.Errorf("send github request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return nil, fmt.Errorf("github returned retryable status: %s", resp.Status)
		}

		return nil, retry.Unrecoverable(fmt.Errorf("github returned unexpected status: %s", resp.Status))
	}

	var issues []IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, retry.Unrecoverable(fmt.Errorf("decode github response: %w", err))
	}

	return &issuesPageResponse{
		issues:     issues,
		linkHeader: resp.Header.Get("Link"),
	}, nil
}

func buildPreview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= previewLimit {
		return text
	}

	textRunes := []rune(text)

	return string(textRunes[:previewLimit])
}
