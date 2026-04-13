package sender

import "context"

// Struct for messages with updates info
type UpdateMessage struct {
	ID          int64
	URL         string
	Description string
	TgChatIDs   []int64
}

// Struct for errors that occurred during updates processing
type ProblemsMessage struct {
	ID          int64
	Description string
	TgChatIDs   []int64
}

// Interface for sending messages to bot
type MessageSender interface {
	SendUpdate(ctx context.Context, msg UpdateMessage) error
	SendProblems(ctx context.Context, msg ProblemsMessage) error
}
