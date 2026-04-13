package scrappertest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
)

func TestScrapperHTTP_AddLink_PersistsEntitiesAndReturnsSavedData(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)

		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestSubscriptionService(t, db)

		server := newScrapperTestServer(subscriptionService)
		defer server.Close()

		client := scrapperhttp.NewClient(server.URL, server.Client())

		const chatID int64 = 777
		const trackedURL = "https://github.com/user/repo"
		wantTags := []string{"backend", "go"}

		err := client.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat over http: %v", err)
		}

		// act
		addedLink, err := client.AddLink(ctx, chatID, scrapperhttp.AddLinkRequest{
			Link: trackedURL,
			Tags: wantTags,
		})
		if err != nil {
			t.Fatalf("add link over http: %v", err)
		}

		listResponse, err := client.ListLinks(ctx, chatID)
		if err != nil {
			t.Fatalf("list links over http: %v", err)
		}

		// assert
		links, err := subscriptionService.ListLinksAll(ctx, chatID)
		if err != nil {
			t.Fatalf("list links through service: %v", err)
		}

		trackedURLs, err := subscriptionService.ListTrackedURLsAll(ctx)
		if err != nil {
			t.Fatalf("list tracked urls through service: %v", err)
		}

		chatIDs, err := subscriptionService.ListChatIDsAll(ctx, trackedURL)
		if err != nil {
			t.Fatalf("list chat ids through service: %v", err)
		}

		gotTags, err := subscriptionService.ListTagsAll(ctx, chatID, trackedURL)
		if err != nil {
			t.Fatalf("list tags through service: %v", err)
		}

		if addedLink.URL != trackedURL {
			t.Errorf("unexpected added link url: got %q, want %q", addedLink.URL, trackedURL)
		}

		slices.Sort(addedLink.Tags)
		sortedWantTags := slices.Clone(wantTags)
		slices.Sort(sortedWantTags)
		if !slices.Equal(addedLink.Tags, sortedWantTags) {
			t.Errorf("unexpected added link tags: got %v, want %v", addedLink.Tags, sortedWantTags)
		}

		if addedLink.ID <= 0 {
			t.Errorf("expected positive link id, got %d", addedLink.ID)
		}

		if len(links) != 1 {
			t.Errorf("unexpected links count in service: got %d, want %d", len(links), 1)
		}

		if len(links) == 1 {
			if links[0].URL != trackedURL {
				t.Errorf("unexpected stored link url: got %q, want %q", links[0].URL, trackedURL)
			}

			slices.Sort(links[0].Tags)
			if !slices.Equal(links[0].Tags, sortedWantTags) {
				t.Errorf("unexpected stored link tags: got %v, want %v", links[0].Tags, sortedWantTags)
			}
		}

		lastUpdated, ok := trackedURLs[trackedURL]
		if !ok {
			t.Errorf("tracked url %q not found in tracked urls map", trackedURL)
		} else if lastUpdated.IsZero() {
			t.Errorf("expected non-zero last updated for tracked url %q", trackedURL)
		}

		if len(chatIDs) != 1 {
			t.Errorf("unexpected chat ids count: got %d, want %d", len(chatIDs), 1)
		}

		if len(chatIDs) == 1 && chatIDs[0] != chatID {
			t.Errorf("unexpected chat id for tracked url: got %d, want %d", chatIDs[0], chatID)
		}

		slices.Sort(gotTags)
		if !slices.Equal(gotTags, sortedWantTags) {
			t.Errorf("unexpected stored tags: got %v, want %v", gotTags, sortedWantTags)
		}

		if listResponse.Size != 1 {
			t.Errorf("unexpected list response size: got %d, want %d", listResponse.Size, 1)
		}

		if len(listResponse.Links) != 1 {
			t.Errorf("unexpected list response links count: got %d, want %d", len(listResponse.Links), 1)
		}

		if len(listResponse.Links) == 1 {
			gotLink := listResponse.Links[0]

			if gotLink.URL != trackedURL {
				t.Errorf("unexpected listed link url: got %q, want %q", gotLink.URL, trackedURL)
			}

			slices.Sort(gotLink.Tags)
			if !slices.Equal(gotLink.Tags, sortedWantTags) {
				t.Errorf("unexpected listed link tags: got %v, want %v", gotLink.Tags, sortedWantTags)
			}

			if gotLink.ID != addedLink.ID {
				t.Errorf("unexpected listed link id: got %d, want %d", gotLink.ID, addedLink.ID)
			}
		}
	})
}

func newScrapperTestServer(subscriptions *service.SubscriptionService) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("/tg-chat/", scrapperhttp.NewTgChatHandler(subscriptions))
	mux.Handle("/links", scrapperhttp.NewLinksHandler(subscriptions))
	return httptest.NewServer(mux)
}
