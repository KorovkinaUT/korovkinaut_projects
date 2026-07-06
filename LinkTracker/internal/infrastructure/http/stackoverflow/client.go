package stackoverflowhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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

// for getting responses in HTML
var htmlTagRegexp = regexp.MustCompile(`<[^>]*>`)

// Client for StackOverflow updates requests
type Client struct {
	baseURL        string
	httpClient     *http.Client
	retryCfg       *config.RetryConfig
	circuitBreaker *gobreaker.CircuitBreaker
}

type QuestionsResponse struct {
	Items []QuestionResponse `json:"items"`
}

type QuestionResponse struct {
	Title string `json:"title"`
}

type OwnerResponse struct {
	DisplayName string `json:"display_name"`
}

type AnswersResponse struct {
	Items   []AnswerResponse `json:"items"`
	HasMore bool             `json:"has_more"`
}

type AnswerResponse struct {
	CreationDate int64         `json:"creation_date"`
	Body         string        `json:"body"`
	Owner        OwnerResponse `json:"owner"`
}

type CommentsResponse struct {
	Items   []CommentResponse `json:"items"`
	HasMore bool              `json:"has_more"`
}

type CommentResponse struct {
	CreationDate int64         `json:"creation_date"`
	Body         string        `json:"body"`
	Owner        OwnerResponse `json:"owner"`
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
		circuitBreaker: httpinfra.NewCircuitBreaker("stackoverflow-http-client", cbCfg),
	}
}

func (c *Client) GetQuestionEvents(
	ctx context.Context,
	questionID int64,
	since time.Time,
) ([]update.StackOverflowEvent, error) {
	questionTitle, err := c.getQuestionTitle(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("get question title: %w", err)
	}

	answerEvents, err := c.getAnswerEvents(ctx, questionID, questionTitle, since)
	if err != nil {
		return nil, fmt.Errorf("get answer events: %w", err)
	}

	commentEvents, err := c.getCommentEvents(ctx, questionID, questionTitle, since)
	if err != nil {
		return nil, fmt.Errorf("get comment events: %w", err)
	}

	events := make([]update.StackOverflowEvent, 0, len(answerEvents)+len(commentEvents))
	events = append(events, answerEvents...)
	events = append(events, commentEvents...)

	return events, nil
}

func (c *Client) getQuestionTitle(ctx context.Context, questionID int64) (string, error) {
	var result string

	_, err := c.circuitBreaker.Execute(func() (any, error) {
		err := retry.Do(
			func() error {
				title, err := c.getQuestionTitleOnce(ctx, questionID)
				if err != nil {
					return err
				}

				result = title

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
		return "", err
	}

	return result, nil
}

func (c *Client) getQuestionTitleOnce(ctx context.Context, questionID int64) (string, error) {
	params := url.Values{}
	params.Set("site", "stackoverflow")

	endpoint := fmt.Sprintf(
		"%s/questions/%d?%s",
		c.baseURL,
		questionID,
		params.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", retry.Unrecoverable(fmt.Errorf("build stackoverflow question request: %w", err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", retry.Unrecoverable(fmt.Errorf("send stackoverflow question request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return "", fmt.Errorf("stackoverflow returned retryable status: %s", resp.Status)
		}

		return "", retry.Unrecoverable(fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status))
	}

	var questions QuestionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&questions); err != nil {
		return "", retry.Unrecoverable(fmt.Errorf("decode stackoverflow question response: %w", err))
	}

	if len(questions.Items) == 0 {
		return "", retry.Unrecoverable(fmt.Errorf("stackoverflow question not found"))
	}

	return questions.Items[0].Title, nil
}

func (c *Client) getAnswerEvents(
	ctx context.Context,
	questionID int64,
	questionTitle string,
	since time.Time,
) ([]update.StackOverflowEvent, error) {
	params := url.Values{}
	params.Set("site", "stackoverflow")
	params.Set("sort", "creation")
	params.Set("order", "desc")
	params.Set("filter", "withbody")
	params.Set("fromdate", strconv.FormatInt(since.Unix(), 10))
	params.Set("pagesize", "100")

	events := make([]update.StackOverflowEvent, 0)
	page := 1

	for {
		params.Set("page", strconv.Itoa(page))

		endpoint := fmt.Sprintf(
			"%s/questions/%d/answers?%s",
			c.baseURL,
			questionID,
			params.Encode(),
		)

		answers, err := c.getAnswersPage(ctx, endpoint)
		if err != nil {
			return nil, err
		}

		for _, answer := range answers.Items {
			events = append(events, update.StackOverflowEvent{
				Type:          update.StackOverflowEventAnswer,
				QuestionTitle: questionTitle,
				Username:      answer.Owner.DisplayName,
				CreationTime:  time.Unix(answer.CreationDate, 0).UTC(),
				Preview:       buildPreview(answer.Body),
			})
		}

		if !answers.HasMore {
			break
		}

		page++
	}

	return events, nil
}

func (c *Client) getAnswersPage(ctx context.Context, endpoint string) (AnswersResponse, error) {
	var result AnswersResponse

	_, err := c.circuitBreaker.Execute(func() (any, error) {
		err := retry.Do(
			func() error {
				answers, err := c.getAnswersPageOnce(ctx, endpoint)
				if err != nil {
					return err
				}

				result = answers

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
		return AnswersResponse{}, err
	}

	return result, nil
}

func (c *Client) getAnswersPageOnce(ctx context.Context, endpoint string) (AnswersResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return AnswersResponse{}, retry.Unrecoverable(fmt.Errorf("build stackoverflow answer request: %w", err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AnswersResponse{}, retry.Unrecoverable(fmt.Errorf("send stackoverflow answer request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return AnswersResponse{}, fmt.Errorf("stackoverflow returned retryable status: %s", resp.Status)
		}

		return AnswersResponse{}, retry.Unrecoverable(fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status))
	}

	var answers AnswersResponse
	if err := json.NewDecoder(resp.Body).Decode(&answers); err != nil {
		return AnswersResponse{}, retry.Unrecoverable(fmt.Errorf("decode stackoverflow answers response: %w", err))
	}

	return answers, nil
}

func (c *Client) getCommentEvents(
	ctx context.Context,
	questionID int64,
	questionTitle string,
	since time.Time,
) ([]update.StackOverflowEvent, error) {
	params := url.Values{}
	params.Set("site", "stackoverflow")
	params.Set("sort", "creation")
	params.Set("order", "desc")
	params.Set("filter", "withbody")
	params.Set("fromdate", strconv.FormatInt(since.Unix(), 10))
	params.Set("pagesize", "100")

	events := make([]update.StackOverflowEvent, 0)
	page := 1

	for {
		params.Set("page", strconv.Itoa(page))

		endpoint := fmt.Sprintf(
			"%s/questions/%d/comments?%s",
			c.baseURL,
			questionID,
			params.Encode(),
		)

		comments, err := c.getCommentsPage(ctx, endpoint)
		if err != nil {
			return nil, err
		}

		for _, comment := range comments.Items {
			events = append(events, update.StackOverflowEvent{
				Type:          update.StackOverflowEventComment,
				QuestionTitle: questionTitle,
				Username:      comment.Owner.DisplayName,
				CreationTime:  time.Unix(comment.CreationDate, 0).UTC(),
				Preview:       buildPreview(comment.Body),
			})
		}

		if !comments.HasMore {
			break
		}

		page++
	}

	return events, nil
}

func (c *Client) getCommentsPage(ctx context.Context, endpoint string) (CommentsResponse, error) {
	var result CommentsResponse

	_, err := c.circuitBreaker.Execute(func() (any, error) {
		err := retry.Do(
			func() error {
				comments, err := c.getCommentsPageOnce(ctx, endpoint)
				if err != nil {
					return err
				}

				result = comments

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
		return CommentsResponse{}, err
	}

	return result, nil
}

func (c *Client) getCommentsPageOnce(ctx context.Context, endpoint string) (CommentsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return CommentsResponse{}, retry.Unrecoverable(fmt.Errorf("build stackoverflow comment request: %w", err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return CommentsResponse{}, retry.Unrecoverable(fmt.Errorf("send stackoverflow comment request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if c.retryCfg.IsRetryableStatus(resp.StatusCode) {
			return CommentsResponse{}, fmt.Errorf("stackoverflow returned retryable status: %s", resp.Status)
		}

		return CommentsResponse{}, retry.Unrecoverable(fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status))
	}

	var comments CommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return CommentsResponse{}, retry.Unrecoverable(fmt.Errorf("decode stackoverflow comments response: %w", err))
	}

	return comments, nil
}

func buildPreview(text string) string {
	// response body is in HTML format
	text = htmlTagRegexp.ReplaceAllString(text, " ")
	text = strings.Join(strings.Fields(text), " ")

	if len(text) <= previewLimit {
		return text
	}

	textRunes := []rune(text)
	return string(textRunes[:previewLimit])
}
