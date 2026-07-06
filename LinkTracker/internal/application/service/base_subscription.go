package service

import (
	"context"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

// Service to get data from repository
type BaseSubscriptionService struct {
	chats         repository.ChatRepository
	subscriptions repository.SubscriptionRepository
}

var _ SubscriptionService = (*BaseSubscriptionService)(nil)

func NewBaseSubscriptionService(
	chats repository.ChatRepository,
	subscriptions repository.SubscriptionRepository,
) *BaseSubscriptionService {
	return &BaseSubscriptionService{
		chats:         chats,
		subscriptions: subscriptions,
	}
}

func (s *BaseSubscriptionService) RegisterChat(ctx context.Context, chatID int64) error {
	return s.chats.Register(ctx, chatID)
}

func (s *BaseSubscriptionService) DeleteChat(ctx context.Context, chatID int64) error {
	return s.chats.Delete(ctx, chatID)
}

func (s *BaseSubscriptionService) AddLink(
	ctx context.Context,
	chatID int64,
	url string,
	tags []string,
) (domain.RepositoryLink, error) {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return domain.RepositoryLink{}, err
	}
	if !exists {
		return domain.RepositoryLink{}, repository.ErrChatNotFound
	}

	if err := s.subscriptions.AddLink(ctx, chatID, url, tags); err != nil {
		return domain.RepositoryLink{}, err
	}

	return s.subscriptions.GetLink(ctx, chatID, url)
}

func (s *BaseSubscriptionService) RemoveLink(ctx context.Context, chatID int64, url string) (domain.RepositoryLink, error) {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return domain.RepositoryLink{}, err
	}
	if !exists {
		return domain.RepositoryLink{}, repository.ErrChatNotFound
	}

	link, err := s.subscriptions.GetLink(ctx, chatID, url)
	if err != nil {
		return domain.RepositoryLink{}, err
	}

	if err := s.subscriptions.RemoveLink(ctx, chatID, url); err != nil {
		return domain.RepositoryLink{}, err
	}

	return link, nil
}

func (s *BaseSubscriptionService) ListLinks(
	ctx context.Context,
	chatID int64,
	limit int64,
	offset int64,
) ([]domain.RepositoryLink, error) {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrChatNotFound
	}

	return s.subscriptions.ListLinks(ctx, chatID, limit, offset)
}

func (s *BaseSubscriptionService) ListLinksAll(ctx context.Context, chatID int64) ([]domain.RepositoryLink, error) {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrChatNotFound
	}

	const limit int64 = 100
	var offset int64 = 0

	result := make([]domain.RepositoryLink, 0)

	for {
		batch, err := s.subscriptions.ListLinks(ctx, chatID, limit, offset)
		if err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			break
		}

		result = append(result, batch...)
		offset += limit
	}

	return result, nil
}

func (s *BaseSubscriptionService) ListChatIDs(ctx context.Context, url string, limit int64, offset int64) ([]int64, error) {
	return s.subscriptions.ListChatIDs(ctx, url, limit, offset)
}

func (s *BaseSubscriptionService) ListChatIDsAll(ctx context.Context, url string) ([]int64, error) {
	const limit int64 = 100
	var offset int64 = 0

	result := make([]int64, 0)

	for {
		batch, err := s.subscriptions.ListChatIDs(ctx, url, limit, offset)
		if err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			break
		}

		result = append(result, batch...)
		offset += limit
	}

	return result, nil
}

func (s *BaseSubscriptionService) ListTrackedURLs(ctx context.Context, limit int64, offset int64) (map[string]time.Time, error) {
	return s.subscriptions.ListTrackedURLs(ctx, limit, offset)
}

func (s *BaseSubscriptionService) ListTrackedURLsAll(ctx context.Context) (map[string]time.Time, error) {
	const limit int64 = 100
	var offset int64 = 0

	result := make(map[string]time.Time)

	for {
		batch, err := s.subscriptions.ListTrackedURLs(ctx, limit, offset)
		if err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			break
		}

		for k, v := range batch {
			result[k] = v
		}

		offset += limit
	}

	return result, nil
}

func (s *BaseSubscriptionService) UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error {
	return s.subscriptions.UpdateLastUpdated(ctx, url, updatedAt)
}

func (s *BaseSubscriptionService) AddTag(ctx context.Context, chatID int64, url string, tag string) error {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return repository.ErrChatNotFound
	}

	return s.subscriptions.AddTag(ctx, chatID, url, tag)
}

func (s *BaseSubscriptionService) RemoveTag(ctx context.Context, chatID int64, url string, tag string) error {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return repository.ErrChatNotFound
	}

	return s.subscriptions.RemoveTag(ctx, chatID, url, tag)
}

func (s *BaseSubscriptionService) ListTags(
	ctx context.Context,
	chatID int64,
	url string,
	limit int64,
	offset int64,
) ([]string, error) {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrChatNotFound
	}

	return s.subscriptions.ListTags(ctx, chatID, url, limit, offset)
}

func (s *BaseSubscriptionService) ListTagsAll(ctx context.Context, chatID int64, url string) ([]string, error) {
	exists, err := s.chats.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrChatNotFound
	}

	const batchSize int64 = 100
	var offset int64 = 0

	result := make([]string, 0)

	for {
		batch, err := s.subscriptions.ListTags(ctx, chatID, url, batchSize, offset)
		if err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			break
		}

		result = append(result, batch...)
		offset += batchSize
	}

	return result, nil
}
