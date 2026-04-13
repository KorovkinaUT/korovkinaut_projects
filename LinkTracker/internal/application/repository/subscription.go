package repository

import (
	"context"
	"errors"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

var (
	ErrLinkAlreadyTracked = errors.New("link already tracked")
	ErrLinkNotFound       = errors.New("link not found")
	ErrTagAlreadyExists   = errors.New("tag already exists")
	ErrTagNotFound        = errors.New("tag not found")
)

type SubscriptionRepository interface {
	AddLink(ctx context.Context, chatID int64, url string, tags []string) error
	RemoveLink(ctx context.Context, chatID int64, url string) error
	GetLink(ctx context.Context, chatID int64, url string) (domain.RepositoryLink, error)
	ListLinks(ctx context.Context, chatID int64, limit int64, offset int64) ([]domain.RepositoryLink, error)
	ListChatIDs(ctx context.Context, url string, limit int64, offset int64) ([]int64, error)

	ListTrackedURLs(ctx context.Context, limit int64, offset int64) (map[string]time.Time, error)
	UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error

	AddTag(ctx context.Context, chatID int64, url string, tag string) error
	RemoveTag(ctx context.Context, chatID int64, url string, tag string) error
	ListTags(ctx context.Context, chatID int64, url string, limit int64, offset int64) ([]string, error)
}
