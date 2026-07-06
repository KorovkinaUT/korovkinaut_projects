package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

// Adds cache checking over base subscription service
type SubscriptionServiceWithCache struct {
	base   SubscriptionService
	cache  ListCache
	logger *slog.Logger
}

var _ SubscriptionService = (*SubscriptionServiceWithCache)(nil)

func NewSubscriptionServiceWithCache(
	base SubscriptionService,
	cache ListCache,
	logger *slog.Logger,
) *SubscriptionServiceWithCache {
	return &SubscriptionServiceWithCache{
		base:   base,
		cache:  cache,
		logger: logger,
	}
}

func (s *SubscriptionServiceWithCache) RegisterChat(ctx context.Context, chatID int64) error {
	return s.base.RegisterChat(ctx, chatID)
}

func (s *SubscriptionServiceWithCache) DeleteChat(ctx context.Context, chatID int64) error {
	if err := s.base.DeleteChat(ctx, chatID); err != nil {
		return err
	}

	s.invalidate(ctx, chatID)

	return nil
}

func (s *SubscriptionServiceWithCache) AddLink(
	ctx context.Context,
	chatID int64,
	url string,
	tags []string,
) (domain.RepositoryLink, error) {
	link, err := s.base.AddLink(ctx, chatID, url, tags)
	if err != nil {
		return domain.RepositoryLink{}, err
	}

	s.invalidate(ctx, chatID)

	return link, nil
}

func (s *SubscriptionServiceWithCache) RemoveLink(ctx context.Context, chatID int64, url string) (domain.RepositoryLink, error) {
	link, err := s.base.RemoveLink(ctx, chatID, url)
	if err != nil {
		return domain.RepositoryLink{}, err
	}

	s.invalidate(ctx, chatID)

	return link, nil
}

func (s *SubscriptionServiceWithCache) ListLinks(
	ctx context.Context,
	chatID int64,
	limit int64,
	offset int64,
) ([]domain.RepositoryLink, error) {
	return s.base.ListLinks(ctx, chatID, limit, offset)
}

func (s *SubscriptionServiceWithCache) ListLinksAll(ctx context.Context, chatID int64) ([]domain.RepositoryLink, error) {
	cached, err := s.cache.Get(ctx, chatID)
	if err != nil {
		s.warn("failed to get links from cache", chatID, err)
	} else if cached != nil {
		var links []domain.RepositoryLink
		if err := json.Unmarshal(cached, &links); err == nil {
			return links, nil
		}

		s.warn("failed to unmarshal cached links", chatID, err)
		s.invalidate(ctx, chatID)
	}

	links, err := s.base.ListLinksAll(ctx, chatID)
	if err != nil {
		return nil, err
	}

	value, err := json.Marshal(links)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, chatID, value); err != nil {
		s.warn("failed to save links to cache", chatID, err)
	}

	return links, nil
}

func (s *SubscriptionServiceWithCache) ListChatIDs(ctx context.Context, url string, limit int64, offset int64) ([]int64, error) {
	return s.base.ListChatIDs(ctx, url, limit, offset)
}

func (s *SubscriptionServiceWithCache) ListChatIDsAll(ctx context.Context, url string) ([]int64, error) {
	return s.base.ListChatIDsAll(ctx, url)
}

func (s *SubscriptionServiceWithCache) ListTrackedURLs(ctx context.Context, limit int64, offset int64) (map[string]time.Time, error) {
	return s.base.ListTrackedURLs(ctx, limit, offset)
}

func (s *SubscriptionServiceWithCache) ListTrackedURLsAll(ctx context.Context) (map[string]time.Time, error) {
	return s.base.ListTrackedURLsAll(ctx)
}

func (s *SubscriptionServiceWithCache) UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error {
	return s.base.UpdateLastUpdated(ctx, url, updatedAt)
}

func (s *SubscriptionServiceWithCache) AddTag(ctx context.Context, chatID int64, url string, tag string) error {
	if err := s.base.AddTag(ctx, chatID, url, tag); err != nil {
		return err
	}

	s.invalidate(ctx, chatID)

	return nil
}

func (s *SubscriptionServiceWithCache) RemoveTag(ctx context.Context, chatID int64, url string, tag string) error {
	if err := s.base.RemoveTag(ctx, chatID, url, tag); err != nil {
		return err
	}

	s.invalidate(ctx, chatID)

	return nil
}

func (s *SubscriptionServiceWithCache) ListTags(
	ctx context.Context,
	chatID int64,
	url string,
	limit int64,
	offset int64,
) ([]string, error) {
	return s.base.ListTags(ctx, chatID, url, limit, offset)
}

func (s *SubscriptionServiceWithCache) ListTagsAll(ctx context.Context, chatID int64, url string) ([]string, error) {
	return s.base.ListTagsAll(ctx, chatID, url)
}

func (s *SubscriptionServiceWithCache) invalidate(ctx context.Context, chatID int64) {
	if err := s.cache.Delete(ctx, chatID); err != nil {
		s.warn("failed to invalidate GET /links cache", chatID, err)
	}
}

func (s *SubscriptionServiceWithCache) warn(msg string, chatID int64, err error) {
	if s.logger == nil {
		return
	}

	s.logger.Warn(msg, "chat_id", chatID, "error", err)
}
