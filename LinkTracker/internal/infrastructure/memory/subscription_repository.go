package memory

import (
	"maps"
	"slices"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

// Implementation of SubscriptionRepository Interface
// Stores chats and links matching
type SubscriptionRepository struct {
	mu sync.RWMutex

	nextLinkID int64 // to get next link id for responce api

	linksByChat      map[int64]map[string]domain.RepositoryLink
	chatsByLink      map[string]map[int64]struct{} // for sending update
	lastUpdatedByURL map[string]time.Time          // for checking updates
}

func NewSubscriptionRepository() *SubscriptionRepository {
	return &SubscriptionRepository{
		nextLinkID:       1,
		linksByChat:      make(map[int64]map[string]domain.RepositoryLink),
		chatsByLink:      make(map[string]map[int64]struct{}),
		lastUpdatedByURL: make(map[string]time.Time),
	}
}

func (r *SubscriptionRepository) AddLink(chatID int64, url string, tags []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.linksByChat[chatID]; !exists {
		r.linksByChat[chatID] = make(map[string]domain.RepositoryLink)
	}

	if _, exists := r.linksByChat[chatID][url]; exists {
		return repository.ErrLinkAlreadyTracked
	}

	link := domain.RepositoryLink{
		ID:   r.nextLinkID,
		URL:  url,
		Tags: slices.Clone(tags),
	}
	r.nextLinkID++

	r.linksByChat[chatID][url] = link

	if _, exists := r.chatsByLink[url]; !exists {
		r.chatsByLink[url] = make(map[int64]struct{})
		r.lastUpdatedByURL[url] = time.Now()
	}
	r.chatsByLink[url][chatID] = struct{}{}

	return nil
}

func (r *SubscriptionRepository) RemoveLink(chatID int64, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	chatLinks, exists := r.linksByChat[chatID]
	if !exists {
		return repository.ErrLinkNotFound
	}

	if _, exists := chatLinks[url]; !exists {
		return repository.ErrLinkNotFound
	}

	delete(chatLinks, url)
	if len(chatLinks) == 0 {
		delete(r.linksByChat, chatID)
	}

	linkChats, exists := r.chatsByLink[url]
	if exists {
		delete(linkChats, chatID)
		if len(linkChats) == 0 {
			delete(r.chatsByLink, url)
			delete(r.lastUpdatedByURL, url)
		}
	}

	return nil
}

func (r *SubscriptionRepository) GetLink(chatID int64, url string) (domain.RepositoryLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chatLinks, exists := r.linksByChat[chatID]
	if !exists {
		return domain.RepositoryLink{}, repository.ErrLinkNotFound
	}

	link, exists := chatLinks[url]
	if !exists {
		return domain.RepositoryLink{}, repository.ErrLinkNotFound
	}

	return cloneLink(link), nil
}

func (r *SubscriptionRepository) ListLinks(chatID int64) ([]domain.RepositoryLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chatLinks, exists := r.linksByChat[chatID]
	if !exists {
		return []domain.RepositoryLink{}, nil
	}

	result := make([]domain.RepositoryLink, 0, len(chatLinks))
	for _, link := range chatLinks {
		result = append(result, cloneLink(link))
	}

	return result, nil
}

func (r *SubscriptionRepository) ListChatIDs(url string) ([]int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	linkChats, exists := r.chatsByLink[url]
	if !exists {
		return []int64{}, nil
	}

	result := make([]int64, 0, len(linkChats))
	for chatID := range linkChats {
		result = append(result, chatID)
	}

	return result, nil
}

func (r *SubscriptionRepository) ListTrackedURLs() (map[string]time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return maps.Clone(r.lastUpdatedByURL), nil
}

func (r *SubscriptionRepository) UpdateLastUpdated(url string, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.lastUpdatedByURL[url]; !exists {
		return repository.ErrLinkNotFound
	}

	r.lastUpdatedByURL[url] = updatedAt
	return nil
}

// From outside Links can't be changed
func cloneLink(link domain.RepositoryLink) domain.RepositoryLink {
	return domain.RepositoryLink{
		ID:   link.ID,
		URL:  link.URL,
		Tags: slices.Clone(link.Tags),
	}
}

// CompileTime check of methods correctness
var _ repository.SubscriptionRepository = (*SubscriptionRepository)(nil)
