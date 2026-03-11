package updates

import (
	"context"
	"fmt"
	"strconv"
	"time"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
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

func (c *StackOverflowClient) GetUpdatedAt(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
	stackLink, ok := link.(schedulerlink.StackOverflowLink)
	if !ok {
		return time.Time{}, fmt.Errorf("expected stackoverflow link, got %T", link)
	}

	updatedAt, err := c.client.GetQuestionUpdatedAt(ctx, strconv.FormatInt(stackLink.QuestionID, 10))
	if err != nil {
		return time.Time{}, fmt.Errorf("get stackoverflow updated at: %w", err)
	}

	return updatedAt, nil
}
