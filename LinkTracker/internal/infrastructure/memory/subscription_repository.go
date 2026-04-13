package memory

import (
	"context"
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

	nextLinkID int64 // to get next link id for response api

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

func (r *SubscriptionRepository) AddLink(ctx context.Context, chatID int64, url string, tags []string) error {
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

func (r *SubscriptionRepository) RemoveLink(ctx context.Context, chatID int64, url string) error {
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

func (r *SubscriptionRepository) GetLink(ctx context.Context, chatID int64, url string) (domain.RepositoryLink, error) {
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

func (r *SubscriptionRepository) ListLinks(ctx context.Context, chatID int64, limit int64, offset int64) ([]domain.RepositoryLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chatLinks, exists := r.linksByChat[chatID]
	if !exists {
		return []domain.RepositoryLink{}, nil
	}

	links := make([]domain.RepositoryLink, 0, len(chatLinks))
	for _, link := range chatLinks {
		links = append(links, cloneLink(link))
	}

	slices.SortFunc(links, func(a, b domain.RepositoryLink) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	return paginate(links, limit, offset), nil
}

func (r *SubscriptionRepository) ListChatIDs(ctx context.Context, url string, limit int64, offset int64) ([]int64, error) {
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

	slices.Sort(result)

	return paginate(result, limit, offset), nil
}

func (r *SubscriptionRepository) ListTrackedURLs(ctx context.Context, limit int64, offset int64) (map[string]time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	urls := make([]string, 0, len(r.lastUpdatedByURL))
	for url := range r.lastUpdatedByURL {
		urls = append(urls, url)
	}
	slices.Sort(urls)

	urls = paginate(urls, limit, offset)

	result := make(map[string]time.Time, len(urls))
	for _, url := range urls {
		result[url] = r.lastUpdatedByURL[url]
	}

	return result, nil
}

func (r *SubscriptionRepository) UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.lastUpdatedByURL[url]; !exists {
		return repository.ErrLinkNotFound
	}

	r.lastUpdatedByURL[url] = updatedAt
	return nil
}

func (r *SubscriptionRepository) AddTag(ctx context.Context, chatID int64, url string, tag string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	chatLinks, exists := r.linksByChat[chatID]
	if !exists {
		return repository.ErrLinkNotFound
	}

	link, exists := chatLinks[url]
	if !exists {
		return repository.ErrLinkNotFound
	}

	if slices.Contains(link.Tags, tag) {
		return repository.ErrTagAlreadyExists
	}

	link.Tags = append(link.Tags, tag)
	chatLinks[url] = link

	return nil
}

func (r *SubscriptionRepository) RemoveTag(ctx context.Context, chatID int64, url string, tag string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	chatLinks, exists := r.linksByChat[chatID]
	if !exists {
		return repository.ErrLinkNotFound
	}

	link, exists := chatLinks[url]
	if !exists {
		return repository.ErrLinkNotFound
	}

	tagIndex := -1
	for i, existingTag := range link.Tags {
		if existingTag == tag {
			tagIndex = i
			break
		}
	}
	if tagIndex == -1 {
		return repository.ErrTagNotFound
	}

	link.Tags = append(link.Tags[:tagIndex], link.Tags[tagIndex+1:]...)
	chatLinks[url] = link

	return nil
}

func (r *SubscriptionRepository) ListTags(ctx context.Context, chatID int64, url string, limit int64, offset int64) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chatLinks, exists := r.linksByChat[chatID]
	if !exists {
		return nil, repository.ErrLinkNotFound
	}

	link, exists := chatLinks[url]
	if !exists {
		return nil, repository.ErrLinkNotFound
	}

	tags := slices.Clone(link.Tags)

	return paginate(tags, limit, offset), nil
}

// From outside Links can't be changed
func cloneLink(link domain.RepositoryLink) domain.RepositoryLink {
	return domain.RepositoryLink{
		ID:   link.ID,
		URL:  link.URL,
		Tags: slices.Clone(link.Tags),
	}
}

func paginate[T any](items []T, limit int64, offset int64) []T {
	if offset >= int64(len(items)) {
		return []T{}
	}

	end := offset + limit
	if end > int64(len(items)) {
		end = int64(len(items))
	}

	return slices.Clone(items[offset:end])
}

// CompileTime check of methods correctness
var _ repository.SubscriptionRepository = (*SubscriptionRepository)(nil)
