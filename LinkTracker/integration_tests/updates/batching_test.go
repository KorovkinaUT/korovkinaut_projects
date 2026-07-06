package updatestest

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	appupdates "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
)

func TestChecker_BatchProcessing_ProcessesAllBatches(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)
		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestBaseSubscriptionService(t, db)

		const chatID int64 = 501

		urls := []string{
			"https://github.com/user/repo-one",
			"https://github.com/user/repo-two",
			"https://github.com/user/repo-three",
		}

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		lastUpdated := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		newEventTime := time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)

		for _, url := range urls {
			_, err = subscriptionService.AddLink(ctx, chatID, url, []string{"batch"})
			if err != nil {
				t.Fatalf("add link %q: %v", url, err)
			}

			err = subscriptionService.UpdateLastUpdated(ctx, url, lastUpdated)
			if err != nil {
				t.Fatalf("update last updated for %q: %v", url, err)
			}
		}

		githubResponses := map[string]any{
			"/repos/user/repo-one/issues": []map[string]any{
				{
					"title":      "Issue one",
					"body":       "Body one",
					"created_at": newEventTime.Format(time.RFC3339),
					"user": map[string]any{
						"login": "alice",
					},
				},
			},
			"/repos/user/repo-two/issues": []map[string]any{
				{
					"title":      "Issue two",
					"body":       "Body two",
					"created_at": newEventTime.Add(time.Minute).Format(time.RFC3339),
					"user": map[string]any{
						"login": "bob",
					},
				},
			},
			"/repos/user/repo-three/issues": []map[string]any{
				{
					"title":      "Issue three",
					"body":       "Body three",
					"created_at": newEventTime.Add(2 * time.Minute).Format(time.RFC3339),
					"user": map[string]any{
						"login": "carol",
					},
				},
			},
		}

		githubServer := newGitHubBatchServer(t, githubResponses, nil)
		defer githubServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		githubClient := githubhttp.NewClient(githubServer.URL, httpClient, testRetryConfig(), testCircuitBreakerConfig())
		rawSender := newRawUpdateSenderStub()

		checker := appupdates.NewChecker(
			logger,
			2,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			rawSender,
			[]appupdates.LinkClient{
				appupdates.NewGitHubClient(githubClient),
			},
		)

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Fatalf("checker returned error: %v", err)
		}

		if len(rawSender.updates) != len(urls) {
			t.Fatalf("unexpected sent updates count: got %d, want %d", len(rawSender.updates), len(urls))
		}

		if len(rawSender.problems) != 0 {
			t.Errorf("unexpected problem batches count: got %d, want 0", len(rawSender.problems))
		}

		gotByURL := make(map[string]sender.RawUpdateEvents, len(rawSender.updates))
		for _, update := range rawSender.updates {
			gotByURL[update.URL] = update
		}

		expectedTitles := map[string]string{
			"https://github.com/user/repo-one":   "Issue one",
			"https://github.com/user/repo-two":   "Issue two",
			"https://github.com/user/repo-three": "Issue three",
		}
		expectedAuthors := map[string]string{
			"https://github.com/user/repo-one":   "alice",
			"https://github.com/user/repo-two":   "bob",
			"https://github.com/user/repo-three": "carol",
		}

		for _, url := range urls {
			updateMsg, ok := gotByURL[url]
			if !ok {
				t.Errorf("expected raw update for url %q", url)
				continue
			}

			if len(updateMsg.TgChatIDs) != 1 || updateMsg.TgChatIDs[0] != chatID {
				t.Errorf("unexpected chat ids for url %q: got %v, want [%d]", url, updateMsg.TgChatIDs, chatID)
			}

			if len(updateMsg.Events) != 1 {
				t.Errorf("unexpected events count for url %q: got %d, want 1", url, len(updateMsg.Events))
				continue
			}

			event := updateMsg.Events[0]

			if event.Source != string(schedulerlink.TypeGitHub) {
				t.Errorf("unexpected event source for url %q: got %q, want %q", url, event.Source, string(schedulerlink.TypeGitHub))
			}

			if event.Type != "issue" {
				t.Errorf("unexpected event type for url %q: got %q, want %q", url, event.Type, "issue")
			}

			if event.Title != expectedTitles[url] {
				t.Errorf("unexpected event title for url %q: got %q, want %q", url, event.Title, expectedTitles[url])
			}

			if event.Author != expectedAuthors[url] {
				t.Errorf("unexpected event author for url %q: got %q, want %q", url, event.Author, expectedAuthors[url])
			}
		}

		trackedURLs, err := subscriptionService.ListTrackedURLsAll(ctx)
		if err != nil {
			t.Fatalf("list tracked urls: %v", err)
		}

		for _, url := range urls {
			gotUpdatedAt, ok := trackedURLs[url]
			if !ok {
				t.Errorf("tracked url %q not found after checker run", url)
				continue
			}

			if !gotUpdatedAt.After(lastUpdated) {
				t.Errorf("expected last updated to advance for url %q: got %v, last=%v", url, gotUpdatedAt, lastUpdated)
			}
		}
	})
}

func TestChecker_BatchProcessing_PartialFailureDoesNotStopOtherLinks(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)
		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestBaseSubscriptionService(t, db)

		const chatID int64 = 502

		successURL1 := "https://github.com/user/repo-one"
		failedURL := "https://github.com/user/repo-two"
		successURL2 := "https://github.com/user/repo-three"

		urls := []string{successURL1, failedURL, successURL2}

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		lastUpdated := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		newEventTime := time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)

		for _, url := range urls {
			_, err = subscriptionService.AddLink(ctx, chatID, url, []string{"batch"})
			if err != nil {
				t.Fatalf("add link %q: %v", url, err)
			}

			err = subscriptionService.UpdateLastUpdated(ctx, url, lastUpdated)
			if err != nil {
				t.Fatalf("update last updated for %q: %v", url, err)
			}
		}

		githubResponses := map[string]any{
			"/repos/user/repo-one/issues": []map[string]any{
				{
					"title":      "Issue one",
					"body":       "Body one",
					"created_at": newEventTime.Format(time.RFC3339),
					"user": map[string]any{
						"login": "alice",
					},
				},
			},
			"/repos/user/repo-three/issues": []map[string]any{
				{
					"title":      "Issue three",
					"body":       "Body three",
					"created_at": newEventTime.Add(2 * time.Minute).Format(time.RFC3339),
					"user": map[string]any{
						"login": "carol",
					},
				},
			},
		}

		failingPaths := map[string]int{
			"/repos/user/repo-two/issues": http.StatusInternalServerError,
		}

		githubServer := newGitHubBatchServer(t, githubResponses, failingPaths)
		defer githubServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		githubClient := githubhttp.NewClient(githubServer.URL, httpClient, testRetryConfig(), testCircuitBreakerConfig())
		rawSender := newRawUpdateSenderStub()

		checker := appupdates.NewChecker(
			logger,
			2,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			rawSender,
			[]appupdates.LinkClient{
				appupdates.NewGitHubClient(githubClient),
			},
		)

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Fatalf("checker returned error: %v", err)
		}

		if len(rawSender.updates) != 2 {
			t.Fatalf("unexpected successful raw updates count: got %d, want 2", len(rawSender.updates))
		}

		if len(rawSender.problems) != 1 {
			t.Fatalf("unexpected problem batches count: got %d, want 1", len(rawSender.problems))
		}

		gotSuccessURLs := make([]string, 0, len(rawSender.updates))
		for _, updateMsg := range rawSender.updates {
			gotSuccessURLs = append(gotSuccessURLs, updateMsg.URL)

			if len(updateMsg.TgChatIDs) != 1 || updateMsg.TgChatIDs[0] != chatID {
				t.Errorf("unexpected chat ids for successful update %q: got %v, want [%d]", updateMsg.URL, updateMsg.TgChatIDs, chatID)
			}

			if len(updateMsg.Events) != 1 {
				t.Errorf("unexpected events count for update %q: got %d, want 1", updateMsg.URL, len(updateMsg.Events))
				continue
			}

			event := updateMsg.Events[0]
			if event.Source != string(schedulerlink.TypeGitHub) {
				t.Errorf("unexpected event source for url %q: got %q, want %q", updateMsg.URL, event.Source, string(schedulerlink.TypeGitHub))
			}

			if event.Type != "issue" {
				t.Errorf("unexpected event type for url %q: got %q, want %q", updateMsg.URL, event.Type, "issue")
			}
		}

		slices.Sort(gotSuccessURLs)
		wantSuccessURLs := []string{successURL1, successURL2}
		slices.Sort(wantSuccessURLs)
		if !slices.Equal(gotSuccessURLs, wantSuccessURLs) {
			t.Errorf("unexpected successful urls: got %v, want %v", gotSuccessURLs, wantSuccessURLs)
		}

		problems := rawSender.problems[0]
		if len(problems) != 1 {
			t.Fatalf("unexpected problems count: got %d, want 1", len(problems))
		}

		problem := problems[0]

		if problem.URL != failedURL {
			t.Errorf("unexpected problem url: got %q, want %q", problem.URL, failedURL)
		}

		if len(problem.ChatIDs) != 1 || problem.ChatIDs[0] != chatID {
			t.Errorf("unexpected chat ids for problem: got %v, want [%d]", problem.ChatIDs, chatID)
		}

		if !strings.Contains(strings.ToLower(problem.Message), "unexpected status") {
			t.Errorf("problem message must contain failure reason, got %q", problem.Message)
		}

		trackedURLs, err := subscriptionService.ListTrackedURLsAll(ctx)
		if err != nil {
			t.Fatalf("list tracked urls: %v", err)
		}

		for _, url := range []string{successURL1, successURL2} {
			gotUpdatedAt, ok := trackedURLs[url]
			if !ok {
				t.Errorf("tracked url %q not found after checker run", url)
				continue
			}

			if !gotUpdatedAt.After(lastUpdated) {
				t.Errorf("expected last updated to advance for successful url %q: got %v, last=%v", url, gotUpdatedAt, lastUpdated)
			}
		}

		failedUpdatedAt, ok := trackedURLs[failedURL]
		if !ok {
			t.Errorf("tracked url %q not found after checker run", failedURL)
		} else if !failedUpdatedAt.Equal(lastUpdated) {
			t.Errorf("expected failed url updated_at to stay unchanged: got %v, want %v", failedUpdatedAt, lastUpdated)
		}
	})
}

func newGitHubBatchServer(
	t *testing.T,
	successResponses map[string]any,
	failingPaths map[string]int,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected github method: got %s, want %s", r.Method, http.MethodGet)
		}

		if got := r.URL.Query().Get("state"); got != "all" {
			t.Errorf("unexpected github query state: got %q, want %q", got, "all")
		}
		if got := r.URL.Query().Get("sort"); got != "created" {
			t.Errorf("unexpected github query sort: got %q, want %q", got, "created")
		}
		if got := r.URL.Query().Get("direction"); got != "desc" {
			t.Errorf("unexpected github query direction: got %q, want %q", got, "desc")
		}

		if status, ok := failingPaths[r.URL.Path]; ok {
			w.WriteHeader(status)
			return
		}

		response, ok := successResponses[r.URL.Path]
		if !ok {
			t.Errorf("unexpected github path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode github response: %v", err)
		}
	}))
}
