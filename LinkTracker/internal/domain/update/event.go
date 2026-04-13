package update

import "time"

// Interface for new events
type Event interface {
	CreatedAt() time.Time
}
