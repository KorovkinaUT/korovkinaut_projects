package updates

import (
	"context"
	"time"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
)

// Interface for request link update
type LinkClient interface {
	Type() schedulerlink.LinkType
	GetUpdatedAt(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error)
}