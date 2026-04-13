package database_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
)

func TestSubscriptionRepository_AddLinkSavesItInDatabase(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(201)
			url := "https://example.com/a"
			expectedTags := []string{"go", "ml"}

			if err := env.chatRepo.Register(ctx, chatID); err != nil {
				t.Fatalf("register chat: %v", err)
			}

			// act
			err := env.subscriptionRepo.AddLink(ctx, chatID, url, []string{"ml", "go"})
			if err != nil {
				t.Fatalf("add link: %v", err)
			}

			link, err := env.subscriptionRepo.GetLink(ctx, chatID, url)
			if err != nil {
				t.Fatalf("get link: %v", err)
			}

			// assert
			if link.URL != url {
				t.Errorf("expected url %q, got %q", url, link.URL)
			}
			if !slices.Equal(link.Tags, expectedTags) {
				t.Errorf("expected tags %v, got %v", expectedTags, link.Tags)
			}
		})
	}
}

func TestSubscriptionRepository_AddingDuplicateLinkReturnsError(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(202)
			url := "https://example.com/duplicate"

			if err := env.chatRepo.Register(ctx, chatID); err != nil {
				t.Fatalf("register chat: %v", err)
			}
			if err := env.subscriptionRepo.AddLink(ctx, chatID, url, nil); err != nil {
				t.Fatalf("prepare first add link: %v", err)
			}

			// act
			err := env.subscriptionRepo.AddLink(ctx, chatID, url, nil)

			// assert
			if !errors.Is(err, repository.ErrLinkAlreadyTracked) {
				t.Errorf("expected ErrLinkAlreadyTracked, got %v", err)
			}
		})
	}
}

func TestSubscriptionRepository_RemoveLinkDeletesIt(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(203)
			url := "https://example.com/remove"

			if err := env.chatRepo.Register(ctx, chatID); err != nil {
				t.Fatalf("register chat: %v", err)
			}
			if err := env.subscriptionRepo.AddLink(ctx, chatID, url, nil); err != nil {
				t.Fatalf("prepare add link: %v", err)
			}

			// act
			err := env.subscriptionRepo.RemoveLink(ctx, chatID, url)
			if err != nil {
				t.Fatalf("remove link: %v", err)
			}

			_, err = env.subscriptionRepo.GetLink(ctx, chatID, url)

			// assert
			if !errors.Is(err, repository.ErrLinkNotFound) {
				t.Errorf("expected ErrLinkNotFound, got %v", err)
			}
		})
	}
}

func TestSubscriptionRepository_ListLinks(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(204)

			if err := env.chatRepo.Register(ctx, chatID); err != nil {
				t.Fatalf("register chat: %v", err)
			}

			_ = env.subscriptionRepo.AddLink(ctx, chatID, "https://a.com", nil)
			_ = env.subscriptionRepo.AddLink(ctx, chatID, "https://b.com", nil)

			// act
			links, err := env.subscriptionRepo.ListLinks(ctx, chatID, 10, 0)
			if err != nil {
				t.Fatalf("list links: %v", err)
			}

			// assert
			if len(links) != 2 {
				t.Errorf("expected 2 links, got %d", len(links))
			}
		})
	}
}

func TestSubscriptionRepository_ListChatIDs(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			url := "https://shared.com"
			chat1 := int64(205)
			chat2 := int64(206)

			_ = env.chatRepo.Register(ctx, chat1)
			_ = env.chatRepo.Register(ctx, chat2)

			_ = env.subscriptionRepo.AddLink(ctx, chat1, url, nil)
			_ = env.subscriptionRepo.AddLink(ctx, chat2, url, nil)

			// act
			ids, err := env.subscriptionRepo.ListChatIDs(ctx, url, 10, 0)
			if err != nil {
				t.Fatalf("list chat ids: %v", err)
			}

			// assert
			expected := []int64{chat1, chat2}
			if !slices.Equal(ids, expected) {
				t.Errorf("expected %v, got %v", expected, ids)
			}
		})
	}
}

func TestSubscriptionRepository_UpdateLastUpdated(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(207)
			url := "https://update.com"

			_ = env.chatRepo.Register(ctx, chatID)
			_ = env.subscriptionRepo.AddLink(ctx, chatID, url, nil)

			now := time.Now().UTC().Truncate(time.Microsecond)

			// act
			err := env.subscriptionRepo.UpdateLastUpdated(ctx, url, now)
			if err != nil {
				t.Fatalf("update last updated: %v", err)
			}

			tracked, err := env.subscriptionRepo.ListTrackedURLs(ctx, 10, 0)
			if err != nil {
				t.Fatalf("list tracked urls: %v", err)
			}

			// assert
			if !tracked[url].Equal(now) {
				t.Errorf("expected %v, got %v", now, tracked[url])
			}
		})
	}
}

func TestSubscriptionRepository_TagsLifecycle(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(208)
			url := "https://tags.com"

			_ = env.chatRepo.Register(ctx, chatID)
			_ = env.subscriptionRepo.AddLink(ctx, chatID, url, nil)

			// act
			err := env.subscriptionRepo.AddTag(ctx, chatID, url, "go")
			if err != nil {
				t.Fatalf("add tag: %v", err)
			}

			err = env.subscriptionRepo.AddTag(ctx, chatID, url, "go")

			// assert
			if !errors.Is(err, repository.ErrTagAlreadyExists) {
				t.Errorf("expected ErrTagAlreadyExists, got %v", err)
			}

			tags, err := env.subscriptionRepo.ListTags(ctx, chatID, url, 10, 0)
			if err != nil {
				t.Fatalf("list tags: %v", err)
			}

			if !slices.Equal(tags, []string{"go"}) {
				t.Errorf("expected [go], got %v", tags)
			}

			// act
			err = env.subscriptionRepo.RemoveTag(ctx, chatID, url, "go")
			if err != nil {
				t.Fatalf("remove tag: %v", err)
			}

			tags, err = env.subscriptionRepo.ListTags(ctx, chatID, url, 10, 0)
			if err != nil {
				t.Fatalf("list tags after remove: %v", err)
			}

			// assert
			if len(tags) != 0 {
				t.Errorf("expected no tags, got %v", tags)
			}
		})
	}
}
