package update

import "time"

// Interface for new events
type Event interface {
	EventType() string
	EventTitle() string
	EventAuthor() string
	CreatedAt() time.Time
	EventPreview() string
}
