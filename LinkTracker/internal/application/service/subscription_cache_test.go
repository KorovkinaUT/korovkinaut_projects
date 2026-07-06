package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

func TestCachedSubscriptionService_ListLinksAll_CacheHitReturnsCachedValue(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)

	expectedLinks := []domain.RepositoryLink{
		{
			URL:  "https://github.com/test/repo",
			Tags: []string{"go", "backend"},
		},
	}

	base := &fakeSubscriptionService{
		listLinksAllResult: []domain.RepositoryLink{
			{
				URL:  "https://github.com/other/repo",
				Tags: []string{"wrong"},
			},
		},
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo","tags":["go","backend"]}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	actualLinks, err := service.ListLinksAll(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedLinks, actualLinks) {
		t.Errorf("expected links %v, got %v", expectedLinks, actualLinks)
	}
	if base.listLinksAllCalls != 0 {
		t.Errorf("expected base ListLinksAll not to be called, got %d calls", base.listLinksAllCalls)
	}
}

func TestCachedSubscriptionService_ListLinksAll_CacheMissCallsBaseAndSavesToCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)

	expectedLinks := []domain.RepositoryLink{
		{
			URL:  "https://github.com/test/repo",
			Tags: []string{"go"},
		},
	}

	base := &fakeSubscriptionService{
		listLinksAllResult: expectedLinks,
	}

	cache := newFakeListResponseCache()
	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	actualLinks, err := service.ListLinksAll(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedLinks, actualLinks) {
		t.Errorf("expected links %v, got %v", expectedLinks, actualLinks)
	}
	if base.listLinksAllCalls != 1 {
		t.Errorf("expected base ListLinksAll to be called once, got %d calls", base.listLinksAllCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected cache Set to be called once, got %d calls", cache.setCalls)
	}
	if _, ok := cache.values[chatID]; !ok {
		t.Errorf("expected cache value for chatID %d to be saved", chatID)
	}
}

func TestCachedSubscriptionService_ListLinksAll_CacheGetErrorCallsBase(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)

	expectedLinks := []domain.RepositoryLink{
		{
			URL: "https://github.com/test/repo",
		},
	}

	base := &fakeSubscriptionService{
		listLinksAllResult: expectedLinks,
	}

	cache := newFakeListResponseCache()
	cache.getErr = errors.New("cache get failed")

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	actualLinks, err := service.ListLinksAll(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedLinks, actualLinks) {
		t.Errorf("expected links %v, got %v", expectedLinks, actualLinks)
	}
	if base.listLinksAllCalls != 1 {
		t.Errorf("expected base ListLinksAll to be called once, got %d calls", base.listLinksAllCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected cache Set to be called once, got %d calls", cache.setCalls)
	}
}

func TestCachedSubscriptionService_ListLinksAll_InvalidCachedJSONInvalidatesAndCallsBase(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)

	expectedLinks := []domain.RepositoryLink{
		{
			URL: "https://github.com/test/repo",
		},
	}

	base := &fakeSubscriptionService{
		listLinksAllResult: expectedLinks,
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`invalid json`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	actualLinks, err := service.ListLinksAll(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedLinks, actualLinks) {
		t.Errorf("expected links %v, got %v", expectedLinks, actualLinks)
	}
	if base.listLinksAllCalls != 1 {
		t.Errorf("expected base ListLinksAll to be called once, got %d calls", base.listLinksAllCalls)
	}
	if cache.deleteCalls != 1 {
		t.Errorf("expected cache Delete to be called once, got %d calls", cache.deleteCalls)
	}
	if cache.setCalls != 1 {
		t.Errorf("expected cache Set to be called once, got %d calls", cache.setCalls)
	}
}

func TestCachedSubscriptionService_ListLinksAll_BaseErrorReturnsErrorAndDoesNotSaveCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	expectedErr := errors.New("base error")

	base := &fakeSubscriptionService{
		listLinksAllErr: expectedErr,
	}

	cache := newFakeListResponseCache()
	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	actualLinks, err := service.ListLinksAll(ctx, chatID)

	//assert
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if actualLinks != nil {
		t.Errorf("expected nil links, got %v", actualLinks)
	}
	if base.listLinksAllCalls != 1 {
		t.Errorf("expected base ListLinksAll to be called once, got %d calls", base.listLinksAllCalls)
	}
	if cache.setCalls != 0 {
		t.Errorf("expected cache Set not to be called, got %d calls", cache.setCalls)
	}
}

func TestCachedSubscriptionService_AddLink_SuccessInvalidatesCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"
	tags := []string{"go"}

	base := &fakeSubscriptionService{
		addLinkResult: domain.RepositoryLink{
			URL:  url,
			Tags: tags,
		},
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo","tags":["go"]}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	link, err := service.AddLink(ctx, chatID, url, tags)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if link.URL != url {
		t.Errorf("expected link URL %q, got %q", url, link.URL)
	}
	if base.addLinkCalls != 1 {
		t.Errorf("expected base AddLink to be called once, got %d calls", base.addLinkCalls)
	}
	if cache.deleteCalls != 1 {
		t.Errorf("expected cache Delete to be called once, got %d calls", cache.deleteCalls)
	}
	if _, ok := cache.values[chatID]; ok {
		t.Errorf("expected cache value for chatID %d to be deleted", chatID)
	}
}

func TestCachedSubscriptionService_AddLink_ErrorDoesNotInvalidateCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"
	tags := []string{"go"}
	expectedErr := errors.New("add link failed")

	base := &fakeSubscriptionService{
		addLinkErr: expectedErr,
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo","tags":["go"]}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	_, err := service.AddLink(ctx, chatID, url, tags)

	//assert
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if base.addLinkCalls != 1 {
		t.Errorf("expected base AddLink to be called once, got %d calls", base.addLinkCalls)
	}
	if cache.deleteCalls != 0 {
		t.Errorf("expected cache Delete not to be called, got %d calls", cache.deleteCalls)
	}
	if _, ok := cache.values[chatID]; !ok {
		t.Errorf("expected cache value for chatID %d to remain", chatID)
	}
}

func TestCachedSubscriptionService_RemoveLink_SuccessInvalidatesCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"

	base := &fakeSubscriptionService{
		removeLinkResult: domain.RepositoryLink{
			URL: url,
		},
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo"}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	link, err := service.RemoveLink(ctx, chatID, url)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if link.URL != url {
		t.Errorf("expected link URL %q, got %q", url, link.URL)
	}
	if base.removeLinkCalls != 1 {
		t.Errorf("expected base RemoveLink to be called once, got %d calls", base.removeLinkCalls)
	}
	if cache.deleteCalls != 1 {
		t.Errorf("expected cache Delete to be called once, got %d calls", cache.deleteCalls)
	}
	if _, ok := cache.values[chatID]; ok {
		t.Errorf("expected cache value for chatID %d to be deleted", chatID)
	}
}

func TestCachedSubscriptionService_RemoveLink_ErrorDoesNotInvalidateCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"
	expectedErr := errors.New("remove link failed")

	base := &fakeSubscriptionService{
		removeLinkErr: expectedErr,
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo"}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	_, err := service.RemoveLink(ctx, chatID, url)

	//assert
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if base.removeLinkCalls != 1 {
		t.Errorf("expected base RemoveLink to be called once, got %d calls", base.removeLinkCalls)
	}
	if cache.deleteCalls != 0 {
		t.Errorf("expected cache Delete not to be called, got %d calls", cache.deleteCalls)
	}
	if _, ok := cache.values[chatID]; !ok {
		t.Errorf("expected cache value for chatID %d to remain", chatID)
	}
}

func TestCachedSubscriptionService_DeleteChat_SuccessInvalidatesCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)

	base := &fakeSubscriptionService{}
	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo"}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	err := service.DeleteChat(ctx, chatID)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if base.deleteChatCalls != 1 {
		t.Errorf("expected base DeleteChat to be called once, got %d calls", base.deleteChatCalls)
	}
	if cache.deleteCalls != 1 {
		t.Errorf("expected cache Delete to be called once, got %d calls", cache.deleteCalls)
	}
	if _, ok := cache.values[chatID]; ok {
		t.Errorf("expected cache value for chatID %d to be deleted", chatID)
	}
}

func TestCachedSubscriptionService_DeleteChat_ErrorDoesNotInvalidateCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	expectedErr := errors.New("delete chat failed")

	base := &fakeSubscriptionService{
		deleteChatErr: expectedErr,
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo"}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	err := service.DeleteChat(ctx, chatID)

	//assert
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if base.deleteChatCalls != 1 {
		t.Errorf("expected base DeleteChat to be called once, got %d calls", base.deleteChatCalls)
	}
	if cache.deleteCalls != 0 {
		t.Errorf("expected cache Delete not to be called, got %d calls", cache.deleteCalls)
	}
	if _, ok := cache.values[chatID]; !ok {
		t.Errorf("expected cache value for chatID %d to remain", chatID)
	}
}

func TestCachedSubscriptionService_AddTag_SuccessInvalidatesCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"
	tag := "go"

	base := &fakeSubscriptionService{}
	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo","tags":[]}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	err := service.AddTag(ctx, chatID, url, tag)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if base.addTagCalls != 1 {
		t.Errorf("expected base AddTag to be called once, got %d calls", base.addTagCalls)
	}
	if cache.deleteCalls != 1 {
		t.Errorf("expected cache Delete to be called once, got %d calls", cache.deleteCalls)
	}
}

func TestCachedSubscriptionService_AddTag_ErrorDoesNotInvalidateCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"
	tag := "go"
	expectedErr := errors.New("add tag failed")

	base := &fakeSubscriptionService{
		addTagErr: expectedErr,
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo","tags":[]}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	err := service.AddTag(ctx, chatID, url, tag)

	//assert
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if base.addTagCalls != 1 {
		t.Errorf("expected base AddTag to be called once, got %d calls", base.addTagCalls)
	}
	if cache.deleteCalls != 0 {
		t.Errorf("expected cache Delete not to be called, got %d calls", cache.deleteCalls)
	}
}

func TestCachedSubscriptionService_RemoveTag_SuccessInvalidatesCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"
	tag := "go"

	base := &fakeSubscriptionService{}
	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo","tags":["go"]}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	err := service.RemoveTag(ctx, chatID, url, tag)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if base.removeTagCalls != 1 {
		t.Errorf("expected base RemoveTag to be called once, got %d calls", base.removeTagCalls)
	}
	if cache.deleteCalls != 1 {
		t.Errorf("expected cache Delete to be called once, got %d calls", cache.deleteCalls)
	}
}

func TestCachedSubscriptionService_RemoveTag_ErrorDoesNotInvalidateCache(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	url := "https://github.com/test/repo"
	tag := "go"
	expectedErr := errors.New("remove tag failed")

	base := &fakeSubscriptionService{
		removeTagErr: expectedErr,
	}

	cache := newFakeListResponseCache()
	cache.values[chatID] = []byte(`[{"url":"https://github.com/test/repo","tags":["go"]}]`)

	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	err := service.RemoveTag(ctx, chatID, url, tag)

	//assert
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if base.removeTagCalls != 1 {
		t.Errorf("expected base RemoveTag to be called once, got %d calls", base.removeTagCalls)
	}
	if cache.deleteCalls != 0 {
		t.Errorf("expected cache Delete not to be called, got %d calls", cache.deleteCalls)
	}
}

func TestCachedSubscriptionService_ListLinksDelegatesToBase(t *testing.T) {
	//arrange
	ctx := context.Background()
	chatID := int64(1)
	limit := int64(10)
	offset := int64(20)

	expectedLinks := []domain.RepositoryLink{
		{
			URL: "https://github.com/test/repo",
		},
	}

	base := &fakeSubscriptionService{
		listLinksResult: expectedLinks,
	}

	cache := newFakeListResponseCache()
	service := NewSubscriptionServiceWithCache(base, cache, nil)

	//act
	actualLinks, err := service.ListLinks(ctx, chatID, limit, offset)

	//assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedLinks, actualLinks) {
		t.Errorf("expected links %v, got %v", expectedLinks, actualLinks)
	}
	if base.listLinksCalls != 1 {
		t.Errorf("expected base ListLinks to be called once, got %d calls", base.listLinksCalls)
	}
	if cache.getCalls != 0 || cache.setCalls != 0 || cache.deleteCalls != 0 {
		t.Errorf("expected cache not to be used, got get=%d set=%d delete=%d", cache.getCalls, cache.setCalls, cache.deleteCalls)
	}
}

type fakeListResponseCache struct {
	mu sync.Mutex

	values map[int64][]byte

	getErr    error
	setErr    error
	deleteErr error

	getCalls    int
	setCalls    int
	deleteCalls int

	lastGetChatID    int64
	lastSetChatID    int64
	lastDeleteChatID int64
}

func newFakeListResponseCache() *fakeListResponseCache {
	return &fakeListResponseCache{
		values: make(map[int64][]byte),
	}
}

func (c *fakeListResponseCache) Get(ctx context.Context, chatID int64) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.getCalls++
	c.lastGetChatID = chatID

	if c.getErr != nil {
		return nil, c.getErr
	}

	value, ok := c.values[chatID]
	if !ok {
		return nil, nil
	}

	result := make([]byte, len(value))
	copy(result, value)

	return result, nil
}

func (c *fakeListResponseCache) Set(ctx context.Context, chatID int64, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setCalls++
	c.lastSetChatID = chatID

	if c.setErr != nil {
		return c.setErr
	}

	copied := make([]byte, len(value))
	copy(copied, value)
	c.values[chatID] = copied

	return nil
}

func (c *fakeListResponseCache) Delete(ctx context.Context, chatID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteCalls++
	c.lastDeleteChatID = chatID

	if c.deleteErr != nil {
		return c.deleteErr
	}

	delete(c.values, chatID)

	return nil
}

type fakeSubscriptionService struct {
	registerChatErr error
	deleteChatErr   error

	addLinkResult    domain.RepositoryLink
	addLinkErr       error
	removeLinkResult domain.RepositoryLink
	removeLinkErr    error

	listLinksResult    []domain.RepositoryLink
	listLinksErr       error
	listLinksAllResult []domain.RepositoryLink
	listLinksAllErr    error

	listChatIDsResult    []int64
	listChatIDsErr       error
	listChatIDsAllResult []int64
	listChatIDsAllErr    error

	listTrackedURLsResult    map[string]time.Time
	listTrackedURLsErr       error
	listTrackedURLsAllResult map[string]time.Time
	listTrackedURLsAllErr    error

	updateLastUpdatedErr error

	addTagErr    error
	removeTagErr error

	listTagsResult    []string
	listTagsErr       error
	listTagsAllResult []string
	listTagsAllErr    error

	registerChatCalls int
	deleteChatCalls   int
	addLinkCalls      int
	removeLinkCalls   int
	listLinksCalls    int
	listLinksAllCalls int
	addTagCalls       int
	removeTagCalls    int
}

func (s *fakeSubscriptionService) RegisterChat(ctx context.Context, chatID int64) error {
	s.registerChatCalls++
	return s.registerChatErr
}

func (s *fakeSubscriptionService) DeleteChat(ctx context.Context, chatID int64) error {
	s.deleteChatCalls++
	return s.deleteChatErr
}

func (s *fakeSubscriptionService) AddLink(
	ctx context.Context,
	chatID int64,
	url string,
	tags []string,
) (domain.RepositoryLink, error) {
	s.addLinkCalls++
	if s.addLinkErr != nil {
		return domain.RepositoryLink{}, s.addLinkErr
	}

	return s.addLinkResult, nil
}

func (s *fakeSubscriptionService) RemoveLink(
	ctx context.Context,
	chatID int64,
	url string,
) (domain.RepositoryLink, error) {
	s.removeLinkCalls++
	if s.removeLinkErr != nil {
		return domain.RepositoryLink{}, s.removeLinkErr
	}

	return s.removeLinkResult, nil
}

func (s *fakeSubscriptionService) ListLinks(
	ctx context.Context,
	chatID int64,
	limit int64,
	offset int64,
) ([]domain.RepositoryLink, error) {
	s.listLinksCalls++
	if s.listLinksErr != nil {
		return nil, s.listLinksErr
	}

	return s.listLinksResult, nil
}

func (s *fakeSubscriptionService) ListLinksAll(ctx context.Context, chatID int64) ([]domain.RepositoryLink, error) {
	s.listLinksAllCalls++
	if s.listLinksAllErr != nil {
		return nil, s.listLinksAllErr
	}

	return s.listLinksAllResult, nil
}

func (s *fakeSubscriptionService) ListChatIDs(
	ctx context.Context,
	url string,
	limit int64,
	offset int64,
) ([]int64, error) {
	if s.listChatIDsErr != nil {
		return nil, s.listChatIDsErr
	}

	return s.listChatIDsResult, nil
}

func (s *fakeSubscriptionService) ListChatIDsAll(ctx context.Context, url string) ([]int64, error) {
	if s.listChatIDsAllErr != nil {
		return nil, s.listChatIDsAllErr
	}

	return s.listChatIDsAllResult, nil
}

func (s *fakeSubscriptionService) ListTrackedURLs(
	ctx context.Context,
	limit int64,
	offset int64,
) (map[string]time.Time, error) {
	if s.listTrackedURLsErr != nil {
		return nil, s.listTrackedURLsErr
	}

	return s.listTrackedURLsResult, nil
}

func (s *fakeSubscriptionService) ListTrackedURLsAll(ctx context.Context) (map[string]time.Time, error) {
	if s.listTrackedURLsAllErr != nil {
		return nil, s.listTrackedURLsAllErr
	}

	return s.listTrackedURLsAllResult, nil
}

func (s *fakeSubscriptionService) UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error {
	return s.updateLastUpdatedErr
}

func (s *fakeSubscriptionService) AddTag(ctx context.Context, chatID int64, url string, tag string) error {
	s.addTagCalls++
	return s.addTagErr
}

func (s *fakeSubscriptionService) RemoveTag(ctx context.Context, chatID int64, url string, tag string) error {
	s.removeTagCalls++
	return s.removeTagErr
}

func (s *fakeSubscriptionService) ListTags(
	ctx context.Context,
	chatID int64,
	url string,
	limit int64,
	offset int64,
) ([]string, error) {
	if s.listTagsErr != nil {
		return nil, s.listTagsErr
	}

	return s.listTagsResult, nil
}

func (s *fakeSubscriptionService) ListTagsAll(ctx context.Context, chatID int64, url string) ([]string, error) {
	if s.listTagsAllErr != nil {
		return nil, s.listTagsAllErr
	}

	return s.listTagsAllResult, nil
}
