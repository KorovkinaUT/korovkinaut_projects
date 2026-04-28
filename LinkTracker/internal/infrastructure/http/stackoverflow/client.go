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

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

const previewLimit = 200

// for getting responses in HTML
var htmlTagRegexp = regexp.MustCompile(`<[^>]*>`)

// Client for StackOverflow updates requests
type Client struct {
	baseURL    string
	httpClient *http.Client
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

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
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
		return "", fmt.Errorf("build stackoverflow question request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send stackoverflow question request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status)
	}

	var questions QuestionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&questions); err != nil {
		return "", fmt.Errorf("decode stackoverflow question response: %w", err)
	}

	if len(questions.Items) == 0 {
		return "", fmt.Errorf("stackoverflow question not found")
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

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("build stackoverflow answer request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send stackoverflow answer request: %w", err)
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			resp.Body.Close()
			return nil, fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status)
		}

		var answers AnswersResponse
		if err := json.NewDecoder(resp.Body).Decode(&answers); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode stackoverflow answers response: %w", err)
		}
		resp.Body.Close()

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

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("build stackoverflow comment request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send stackoverflow comment request: %w", err)
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			resp.Body.Close()
			return nil, fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status)
		}

		var comments CommentsResponse
		if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode stackoverflow comments response: %w", err)
		}
		resp.Body.Close()

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
