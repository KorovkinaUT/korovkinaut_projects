package updates

import (
	"context"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

// Interface for request link update
type LinkClient interface {
	Type() schedulerlink.LinkType
	GetEvents(ctx context.Context, link schedulerlink.SchedulerLink) ([]update.Event, error)
}
