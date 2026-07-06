package sender

import (
	"context"
	"time"
)

type Event struct {
	Source       string
	Type         string
	Title        string
	Author       string
	Preview      string
	CreationTime time.Time
}

// for sending to agent
type RawUpdateEvents struct {
	URL       string
	Events    []Event
	TgChatIDs []int64
}

type Problem struct {
	URL     string
	Message string
	ChatIDs []int64
}

// Sends updates and problems to agent
type RawUpdateSender interface {
	SendUpdateEvents(ctx context.Context, msg RawUpdateEvents) error
	SendProblems(ctx context.Context, msg []Problem) error
}
