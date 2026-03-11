package service

import (
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

// Service to get data from repository
type SubscriptionService struct {
	chats         repository.ChatRepository
	subscriptions repository.SubscriptionRepository
}

func NewSubscriptionService(
	chats repository.ChatRepository,
	subscriptions repository.SubscriptionRepository,
) *SubscriptionService {
	return &SubscriptionService{
		chats:         chats,
		subscriptions: subscriptions,
	}
}

func (s *SubscriptionService) RegisterChat(chatID int64) error {
	return s.chats.Register(chatID)
}

func (s *SubscriptionService) DeleteChat(chatID int64) error {
	if err := s.chats.Delete(chatID); err != nil {
		return err
	}

	links, err := s.subscriptions.ListLinks(chatID)
	if err != nil {
		return err
	}

	for _, link := range links {
		if err := s.subscriptions.RemoveLink(chatID, link.URL); err != nil {
			return err
		}
	}

	return nil
}

func (s *SubscriptionService) AddLink(chatID int64, url string, tags []string) (domain.RepositoryLink, error) {
	if !s.chats.Exists(chatID) {
		return domain.RepositoryLink{}, repository.ErrChatNotFound
	}

	if err := s.subscriptions.AddLink(chatID, url, tags); err != nil {
		return domain.RepositoryLink{}, err
	}

	return s.subscriptions.GetLink(chatID, url)
}

func (s *SubscriptionService) RemoveLink(chatID int64, url string) (domain.RepositoryLink, error) {
	if !s.chats.Exists(chatID) {
		return domain.RepositoryLink{}, repository.ErrChatNotFound
	}

	link, err := s.subscriptions.GetLink(chatID, url)
	if err != nil {
		return domain.RepositoryLink{}, err
	}

	if err := s.subscriptions.RemoveLink(chatID, url); err != nil {
		return domain.RepositoryLink{}, err
	}

	return link, nil
}

func (s *SubscriptionService) ListLinks(chatID int64) ([]domain.RepositoryLink, error) {
	if !s.chats.Exists(chatID) {
		return nil, repository.ErrChatNotFound
	}

	return s.subscriptions.ListLinks(chatID)
}

func (s *SubscriptionService) ListChatIDs(url string) ([]int64, error) {
	return s.subscriptions.ListChatIDs(url)
}

func (s *SubscriptionService) ListTrackedURLs() (map[string]time.Time, error) {
	return s.subscriptions.ListTrackedURLs()
}

func (s *SubscriptionService) UpdateLastUpdated(url string, updatedAt time.Time) error {
	return s.subscriptions.UpdateLastUpdated(url, updatedAt)
}
