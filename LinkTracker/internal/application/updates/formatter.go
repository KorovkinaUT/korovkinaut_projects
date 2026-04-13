package updates

import (
	"time"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

// Returns text-description of all last updates for URL
type Formatter interface {
	Type() schedulerlink.LinkType
	Format(rawURL string, events []update.Event) (string, error)
}

func formatEventTime(t time.Time) string {
	return t.Format("02 Jan 2006 15:04")
}
