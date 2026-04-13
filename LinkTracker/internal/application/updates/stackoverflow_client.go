package updates

import (
	"context"
	"fmt"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
	stackoverflowhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/stackoverflow"
)

type StackOverflowClient struct {
	client *stackoverflowhttp.Client
}

func NewStackOverflowClient(client *stackoverflowhttp.Client) *StackOverflowClient {
	return &StackOverflowClient{
		client: client,
	}
}

func (c *StackOverflowClient) Type() schedulerlink.LinkType {
	return schedulerlink.TypeStackOverflow
}

func (c *StackOverflowClient) GetEvents(
	ctx context.Context,
	link schedulerlink.SchedulerLink,
) ([]update.Event, error) {
	stackOverflowLink, ok := link.(schedulerlink.StackOverflowLink)
	if !ok {
		return nil, fmt.Errorf("expected stackoverflow link, got %T", link)
	}

	events, err := c.client.GetQuestionEvents(ctx, stackOverflowLink.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("get stackoverflow events: %w", err)
	}

	result := make([]update.Event, 0, len(events))
	for _, event := range events {
		result = append(result, event)
	}

	return result, nil
}
