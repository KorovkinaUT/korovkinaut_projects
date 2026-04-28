package updates

import (
	"context"
	"time"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

// Interface for request link update
type LinkClient interface {
	Type() schedulerlink.LinkType
	GetNewEvents(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error)
}
