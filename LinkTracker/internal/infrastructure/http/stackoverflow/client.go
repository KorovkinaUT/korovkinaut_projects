package stackoverflowhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

const previewLimit = 200

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
	Items []AnswerResponse `json:"items"`
}

type AnswerResponse struct {
	CreationDate int64         `json:"creation_date"`
	Body         string        `json:"body"`
	Owner        OwnerResponse `json:"owner"`
}

type CommentsResponse struct {
	Items []CommentResponse `json:"items"`
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
) ([]update.StackOverflowEvent, error) {
	questionTitle, err := c.getQuestionTitle(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("get question title: %w", err)
	}

	answerEvents, err := c.getAnswerEvents(ctx, questionID, questionTitle)
	if err != nil {
		return nil, fmt.Errorf("get answer events: %w", err)
	}

	commentEvents, err := c.getCommentEvents(ctx, questionID, questionTitle)
	if err != nil {
		return nil, fmt.Errorf("get comment events: %w", err)
	}

	events := make([]update.StackOverflowEvent, 0, len(answerEvents)+len(commentEvents))
	events = append(events, answerEvents...)
	events = append(events, commentEvents...)

	return events, nil
}

func (c *Client) getQuestionTitle(ctx context.Context, questionID int64) (string, error) {
	endpoint := fmt.Sprintf("%s/questions/%d?site=stackoverflow", c.baseURL, questionID)

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
) ([]update.StackOverflowEvent, error) {
	endpoint := fmt.Sprintf(
		"%s/questions/%d/answers?site=stackoverflow&sort=creation&order=desc&filter=withbody",
		c.baseURL,
		questionID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build stackoverflow answer request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send stackoverflow answer request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status)
	}

	var answers AnswersResponse
	if err := json.NewDecoder(resp.Body).Decode(&answers); err != nil {
		return nil, fmt.Errorf("decode stackoverflow answers response: %w", err)
	}

	events := make([]update.StackOverflowEvent, 0, len(answers.Items))
	for _, answer := range answers.Items {
		events = append(events, update.StackOverflowEvent{
			Type:          update.StackOverflowEventAnswer,
			QuestionTitle: questionTitle,
			Username:      answer.Owner.DisplayName,
			CreationTime:  time.Unix(answer.CreationDate, 0).UTC(),
			Preview:       buildPreview(answer.Body),
		})
	}

	return events, nil
}

func (c *Client) getCommentEvents(
	ctx context.Context,
	questionID int64,
	questionTitle string,
) ([]update.StackOverflowEvent, error) {
	endpoint := fmt.Sprintf(
		"%s/questions/%d/comments?site=stackoverflow&sort=creation&order=desc&filter=withbody",
		c.baseURL,
		questionID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build stackoverflow comment request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send stackoverflow comment request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("stackoverflow returned unexpected status: %s", resp.Status)
	}

	var comments CommentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, fmt.Errorf("decode stackoverflow comments response: %w", err)
	}

	events := make([]update.StackOverflowEvent, 0, len(comments.Items))
	for _, comment := range comments.Items {
		events = append(events, update.StackOverflowEvent{
			Type:          update.StackOverflowEventComment,
			QuestionTitle: questionTitle,
			Username:      comment.Owner.DisplayName,
			CreationTime:  time.Unix(comment.CreationDate, 0).UTC(),
			Preview:       buildPreview(comment.Body),
		})
	}

	return events, nil
}

func buildPreview(text string) string {
	// response body in HTML format
	htmlTagRegexp := regexp.MustCompile(`<[^>]*>`)
	text = htmlTagRegexp.ReplaceAllString(text, " ")
	text = strings.Join(strings.Fields(text), " ")

	if len(text) <= previewLimit {
		return text
	}

	return text[:previewLimit]
}
