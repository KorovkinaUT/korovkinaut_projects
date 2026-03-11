package repository

import (
	"errors"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

var (
	ErrLinkAlreadyTracked = errors.New("link already tracked")
	ErrLinkNotFound       = errors.New("link not found")
)

type SubscriptionRepository interface {
	AddLink(chatID int64, url string, tags []string) error
	RemoveLink(chatID int64, url string) error
	GetLink(chatID int64, url string) (domain.RepositoryLink, error)
	ListLinks(chatID int64) ([]domain.RepositoryLink, error)
	ListChatIDs(url string) ([]int64, error)

	ListTrackedURLs() (map[string]time.Time, error)
	UpdateLastUpdated(url string, updatedAt time.Time) error
}
