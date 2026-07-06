package scrappertest

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	valkeyclient "github.com/valkey-io/valkey-go"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
)

func TestScrapperHTTP_ListLinks_CachesResponseAndUsesCache(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)

		helpers.ApplyMigrations(t, db)

		baseService := helpers.NewTestBaseSubscriptionService(t, db)
		countingService := &countingSubscriptionService{
			SubscriptionService: baseService,
		}

		subscriptionService := service.NewSubscriptionServiceWithCache(
			countingService,
			testValkey.Cache,
			nil,
		)

		server := newScrapperTestServer(subscriptionService)
		defer server.Close()

		retryCfg := &config.RetryConfig{
			Attempts:          3,
			Delay:             200 * time.Millisecond,
			RetryableStatuses: []int{500, 502, 503, 504},
		}

		cbCfg := &config.CircuitBreakerConfig{
			SlidingWindowSize:       10,
			FailureRateThreshold:    50,
			CallsInHalfOpen:         5,
			WaitDurationInOpenState: time.Second,
		}

		client := scrapperhttp.NewClient(server.URL, server.Client(), retryCfg, cbCfg)

		const chatID int64 = 9001
		const trackedURL = "https://github.com/user/repo"

		wantTags := []string{"backend", "go"}
		cacheKey := strconv.FormatInt(chatID, 10)

		deleteValkeyKey(t, cacheKey)

		err := client.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat over http: %v", err)
		}

		_, err = client.AddLink(ctx, chatID, scrapperhttp.AddLinkRequest{
			Link: trackedURL,
			Tags: wantTags,
		})
		if err != nil {
			t.Fatalf("add link over http: %v", err)
		}

		deleteValkeyKey(t, cacheKey)

		// act
		firstResponse, err := client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("first list links over http: %v", err)
		}

		// assert
		if countingService.listLinksAllCalls.Load() != 1 {
			t.Errorf(
				"unexpected ListLinksAll calls after first /list: got %d, want %d",
				countingService.listLinksAllCalls.Load(),
				1,
			)
		}

		cachedValue, ok := getValkeyValue(t, cacheKey)
		if !ok {
			t.Fatalf("expected cache value by key %q", cacheKey)
		}

		cachedLinks := unmarshalCachedLinks(t, cachedValue)
		if len(cachedLinks) != 1 {
			t.Errorf("unexpected cached links count: got %d, want %d", len(cachedLinks), 1)
		}

		if len(cachedLinks) == 1 {
			if cachedLinks[0].URL != trackedURL {
				t.Errorf("unexpected cached link url: got %q, want %q", cachedLinks[0].URL, trackedURL)
			}

			slices.Sort(cachedLinks[0].Tags)
			sortedWantTags := slices.Clone(wantTags)
			slices.Sort(sortedWantTags)

			if !slices.Equal(cachedLinks[0].Tags, sortedWantTags) {
				t.Errorf("unexpected cached link tags: got %v, want %v", cachedLinks[0].Tags, sortedWantTags)
			}
		}

		if firstResponse.Size != 1 {
			t.Errorf("unexpected first list response size: got %d, want %d", firstResponse.Size, 1)
		}

		// act
		secondResponse, err := client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("second list links over http: %v", err)
		}

		// assert
		if countingService.listLinksAllCalls.Load() != 1 {
			t.Errorf(
				"expected second /list to use cache without calling service again, got %d ListLinksAll calls",
				countingService.listLinksAllCalls.Load(),
			)
		}

		if secondResponse.Size != firstResponse.Size {
			t.Errorf("unexpected second list response size: got %d, want %d", secondResponse.Size, firstResponse.Size)
		}

		if len(secondResponse.Links) != len(firstResponse.Links) {
			t.Errorf(
				"unexpected second list response links count: got %d, want %d",
				len(secondResponse.Links),
				len(firstResponse.Links),
			)
		}
	})
}

func TestScrapperHTTP_AddLink_InvalidatesCacheAndNextListCallsServiceAgain(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)

		helpers.ApplyMigrations(t, db)

		baseService := helpers.NewTestBaseSubscriptionService(t, db)
		countingService := &countingSubscriptionService{
			SubscriptionService: baseService,
		}

		subscriptionService := service.NewSubscriptionServiceWithCache(
			countingService,
			testValkey.Cache,
			nil,
		)

		server := newScrapperTestServer(subscriptionService)
		defer server.Close()

		retryCfg := &config.RetryConfig{
			Attempts:          3,
			Delay:             200 * time.Millisecond,
			RetryableStatuses: []int{500, 502, 503, 504},
		}

		cbCfg := &config.CircuitBreakerConfig{
			SlidingWindowSize:       10,
			FailureRateThreshold:    50,
			CallsInHalfOpen:         5,
			WaitDurationInOpenState: time.Second,
		}

		client := scrapperhttp.NewClient(server.URL, server.Client(), retryCfg, cbCfg)

		const chatID int64 = 9002
		const firstURL = "https://github.com/user/repo"
		const secondURL = "https://github.com/user/other-repo"

		cacheKey := strconv.FormatInt(chatID, 10)
		deleteValkeyKey(t, cacheKey)

		err := client.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat over http: %v", err)
		}

		_, err = client.AddLink(ctx, chatID, scrapperhttp.AddLinkRequest{
			Link: firstURL,
			Tags: []string{"go"},
		})
		if err != nil {
			t.Fatalf("add first link over http: %v", err)
		}

		deleteValkeyKey(t, cacheKey)

		_, err = client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("first list links over http: %v", err)
		}

		if countingService.listLinksAllCalls.Load() != 1 {
			t.Fatalf(
				"unexpected ListLinksAll calls after first /list: got %d, want %d",
				countingService.listLinksAllCalls.Load(),
				1,
			)
		}

		if _, ok := getValkeyValue(t, cacheKey); !ok {
			t.Fatalf("expected cache value before invalidation")
		}

		_, err = client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("second list links over http: %v", err)
		}

		if countingService.listLinksAllCalls.Load() != 1 {
			t.Fatalf(
				"expected second /list to use cache, got %d ListLinksAll calls",
				countingService.listLinksAllCalls.Load(),
			)
		}

		// act
		_, err = client.AddLink(ctx, chatID, scrapperhttp.AddLinkRequest{
			Link: secondURL,
			Tags: []string{"backend"},
		})
		if err != nil {
			t.Fatalf("add second link over http: %v", err)
		}

		// assert
		if _, ok := getValkeyValue(t, cacheKey); ok {
			t.Errorf("expected cache key %q to be invalidated after AddLink", cacheKey)
		}

		// act
		listResponse, err := client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("list links after cache invalidation over http: %v", err)
		}

		// assert
		if countingService.listLinksAllCalls.Load() != 2 {
			t.Errorf(
				"expected /list after invalidation to call service again, got %d ListLinksAll calls",
				countingService.listLinksAllCalls.Load(),
			)
		}

		if _, ok := getValkeyValue(t, cacheKey); !ok {
			t.Errorf("expected cache value to be written again after /list")
		}

		if listResponse.Size != 2 {
			t.Errorf("unexpected list response size after add: got %d, want %d", listResponse.Size, 2)
		}

		gotURLs := make([]string, 0, len(listResponse.Links))
		for _, link := range listResponse.Links {
			gotURLs = append(gotURLs, link.URL)
		}

		if !slices.Contains(gotURLs, firstURL) {
			t.Errorf("expected list response to contain first url %q, got %v", firstURL, gotURLs)
		}
		if !slices.Contains(gotURLs, secondURL) {
			t.Errorf("expected list response to contain second url %q, got %v", secondURL, gotURLs)
		}
	})
}

func TestScrapperHTTP_RemoveLink_InvalidatesCacheAndNextListCallsServiceAgain(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)

		helpers.ApplyMigrations(t, db)

		baseService := helpers.NewTestBaseSubscriptionService(t, db)
		countingService := &countingSubscriptionService{
			SubscriptionService: baseService,
		}

		subscriptionService := service.NewSubscriptionServiceWithCache(
			countingService,
			testValkey.Cache,
			nil,
		)

		server := newScrapperTestServer(subscriptionService)
		defer server.Close()

		retryCfg := &config.RetryConfig{
			Attempts:          3,
			Delay:             200 * time.Millisecond,
			RetryableStatuses: []int{500, 502, 503, 504},
		}

		cbCfg := &config.CircuitBreakerConfig{
			SlidingWindowSize:       10,
			FailureRateThreshold:    50,
			CallsInHalfOpen:         5,
			WaitDurationInOpenState: time.Second,
		}

		client := scrapperhttp.NewClient(server.URL, server.Client(), retryCfg, cbCfg)

		const chatID int64 = 9003
		const firstURL = "https://github.com/user/repo"
		const secondURL = "https://github.com/user/other-repo"

		cacheKey := strconv.FormatInt(chatID, 10)
		deleteValkeyKey(t, cacheKey)

		err := client.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat over http: %v", err)
		}

		_, err = client.AddLink(ctx, chatID, scrapperhttp.AddLinkRequest{
			Link: firstURL,
			Tags: []string{"go"},
		})
		if err != nil {
			t.Fatalf("add first link over http: %v", err)
		}

		_, err = client.AddLink(ctx, chatID, scrapperhttp.AddLinkRequest{
			Link: secondURL,
			Tags: []string{"backend"},
		})
		if err != nil {
			t.Fatalf("add second link over http: %v", err)
		}

		deleteValkeyKey(t, cacheKey)

		_, err = client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("first list links over http: %v", err)
		}

		if countingService.listLinksAllCalls.Load() != 1 {
			t.Fatalf(
				"unexpected ListLinksAll calls after first /list: got %d, want %d",
				countingService.listLinksAllCalls.Load(),
				1,
			)
		}

		if _, ok := getValkeyValue(t, cacheKey); !ok {
			t.Fatalf("expected cache value before invalidation")
		}

		_, err = client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("second list links over http: %v", err)
		}

		if countingService.listLinksAllCalls.Load() != 1 {
			t.Fatalf(
				"expected second /list to use cache, got %d ListLinksAll calls",
				countingService.listLinksAllCalls.Load(),
			)
		}

		// act
		_, err = client.RemoveLink(ctx, chatID, scrapperhttp.RemoveLinkRequest{
			Link: secondURL,
		})
		if err != nil {
			t.Fatalf("remove link over http: %v", err)
		}

		// assert
		if _, ok := getValkeyValue(t, cacheKey); ok {
			t.Errorf("expected cache key %q to be invalidated after RemoveLink", cacheKey)
		}

		// act
		listResponse, err := client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("list links after cache invalidation over http: %v", err)
		}

		// assert
		if countingService.listLinksAllCalls.Load() != 2 {
			t.Errorf(
				"expected /list after invalidation to call service again, got %d ListLinksAll calls",
				countingService.listLinksAllCalls.Load(),
			)
		}

		if _, ok := getValkeyValue(t, cacheKey); !ok {
			t.Errorf("expected cache value to be written again after /list")
		}

		if listResponse.Size != 1 {
			t.Errorf("unexpected list response size after remove: got %d, want %d", listResponse.Size, 1)
		}

		gotURLs := make([]string, 0, len(listResponse.Links))
		for _, link := range listResponse.Links {
			gotURLs = append(gotURLs, link.URL)
		}

		if !slices.Contains(gotURLs, firstURL) {
			t.Errorf("expected list response to contain remaining url %q, got %v", firstURL, gotURLs)
		}
		if slices.Contains(gotURLs, secondURL) {
			t.Errorf("expected list response not to contain removed url %q, got %v", secondURL, gotURLs)
		}
	})
}

type countingSubscriptionService struct {
	service.SubscriptionService

	listLinksAllCalls atomic.Int64
}

func (s *countingSubscriptionService) ListLinksAll(ctx context.Context, chatID int64) ([]domain.RepositoryLink, error) {
	s.listLinksAllCalls.Add(1)

	return s.SubscriptionService.ListLinksAll(ctx, chatID)
}

func getValkeyValue(t *testing.T, key string) ([]byte, bool) {
	t.Helper()

	result := testValkey.Client.Do(
		context.Background(),
		testValkey.Client.B().
			Get().
			Key(key).
			Build(),
	)

	if err := result.Error(); err != nil {
		if valkeyclient.IsValkeyNil(err) {
			return nil, false
		}

		t.Fatalf("get value from valkey: %v", err)
	}

	value, err := result.AsBytes()
	if err != nil {
		t.Fatalf("convert valkey result to bytes: %v", err)
	}

	return value, true
}

func deleteValkeyKey(t *testing.T, key string) {
	t.Helper()

	err := testValkey.Client.Do(
		context.Background(),
		testValkey.Client.B().
			Del().
			Key(key).
			Build(),
	).Error()
	if err != nil {
		t.Errorf("delete valkey key %q: %v", key, err)
	}
}

func unmarshalCachedLinks(t *testing.T, value []byte) []domain.RepositoryLink {
	t.Helper()

	var links []domain.RepositoryLink
	if err := json.Unmarshal(value, &links); err != nil {
		t.Fatalf("unmarshal cached links: %v; value: %s", err, string(value))
	}

	return links
}
