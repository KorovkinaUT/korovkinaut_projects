package service

import (
	"context"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type SubscriptionService interface {
	RegisterChat(ctx context.Context, chatID int64) error
	DeleteChat(ctx context.Context, chatID int64) error

	AddLink(ctx context.Context, chatID int64, url string, tags []string) (domain.RepositoryLink, error)
	RemoveLink(ctx context.Context, chatID int64, url string) (domain.RepositoryLink, error)

	ListLinks(ctx context.Context, chatID int64, limit int64, offset int64) ([]domain.RepositoryLink, error)
	ListLinksAll(ctx context.Context, chatID int64) ([]domain.RepositoryLink, error)

	ListChatIDs(ctx context.Context, url string, limit int64, offset int64) ([]int64, error)
	ListChatIDsAll(ctx context.Context, url string) ([]int64, error)

	ListTrackedURLs(ctx context.Context, limit int64, offset int64) (map[string]time.Time, error)
	ListTrackedURLsAll(ctx context.Context) (map[string]time.Time, error)

	UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error

	AddTag(ctx context.Context, chatID int64, url string, tag string) error
	RemoveTag(ctx context.Context, chatID int64, url string, tag string) error
	ListTags(ctx context.Context, chatID int64, url string, limit int64, offset int64) ([]string, error)
	ListTagsAll(ctx context.Context, chatID int64, url string) ([]string, error)
}

// Interface for GET /links cache client
type ListCache interface {
	Get(ctx context.Context, chatID int64) ([]byte, error)
	Set(ctx context.Context, chatID int64, value []byte) error
	Delete(ctx context.Context, chatID int64) error
}

func NewSubscriptionService(
	cacheEnabled bool,
	chats repository.ChatRepository,
	subscriptions repository.SubscriptionRepository,
	listCache ListCache,
	logger *slog.Logger,
) SubscriptionService {
	base := NewBaseSubscriptionService(chats, subscriptions)

	if cacheEnabled && listCache != nil {
		return NewSubscriptionServiceWithCache(base, listCache, logger)
	}

	return base
}