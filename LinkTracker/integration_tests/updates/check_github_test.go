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
	"sync"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	appupdates "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
)

func TestChecker_GitHubIssue_SendsFormattedUpdate(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)

		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestBaseSubscriptionService(t, db)

		const chatID int64 = 101
		const trackedURL = "https://github.com/user/repo"

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		_, err = subscriptionService.AddLink(ctx, chatID, trackedURL, []string{"backend"})
		if err != nil {
			t.Fatalf("add link: %v", err)
		}

		lastUpdated := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
		if err != nil {
			t.Fatalf("update last updated: %v", err)
		}

		createdAt := time.Date(2026, 4, 2, 11, 30, 0, 0, time.UTC)
		longBody := strings.Repeat("issue body ", 40)

		githubServer := newGitHubServer(t, []map[string]any{
			{
				"title":      "Fix race in checker",
				"body":       longBody,
				"created_at": createdAt.Format(time.RFC3339),
				"user": map[string]any{
					"login": "octocat",
				},
			},
		})
		defer githubServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		githubClient := githubhttp.NewClient(githubServer.URL, httpClient, testRetryConfig(), testCircuitBreakerConfig())
		rawSender := newRawUpdateSenderStub()

		checker := appupdates.NewChecker(
			logger,
			100,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			rawSender,
			[]appupdates.LinkClient{
				appupdates.NewGitHubClient(githubClient),
			},
		)

		expectedPreview := normalizeAndTrimPreview(longBody, 200)

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Fatalf("checker returned error: %v", err)
		}

		if len(rawSender.problems) != 0 {
			t.Errorf("unexpected problem batches count: got %d, want 0", len(rawSender.problems))
		}

		if len(rawSender.updates) != 1 {
			t.Fatalf("unexpected sent raw updates count: got %d, want 1", len(rawSender.updates))
		}

		updateMsg := rawSender.updates[0]

		if updateMsg.URL != trackedURL {
			t.Errorf("unexpected update url: got %q, want %q", updateMsg.URL, trackedURL)
		}

		slices.Sort(updateMsg.TgChatIDs)
		if !slices.Equal(updateMsg.TgChatIDs, []int64{chatID}) {
			t.Errorf("unexpected chat ids: got %v, want [%d]", updateMsg.TgChatIDs, chatID)
		}

		if len(updateMsg.Events) != 1 {
			t.Fatalf("unexpected events count: got %d, want 1", len(updateMsg.Events))
		}

		event := updateMsg.Events[0]

		if event.Source != string(schedulerlink.TypeGitHub) {
			t.Errorf("unexpected event source: got %q, want %q", event.Source, string(schedulerlink.TypeGitHub))
		}

		if event.Type != "issue" {
			t.Errorf("unexpected event type: got %q, want %q", event.Type, "issue")
		}

		if event.Title != "Fix race in checker" {
			t.Errorf("unexpected event title: got %q, want %q", event.Title, "Fix race in checker")
		}

		if event.Author != "octocat" {
			t.Errorf("unexpected event author: got %q, want %q", event.Author, "octocat")
		}

		if !event.CreationTime.Equal(createdAt) {
			t.Errorf("unexpected event creation time: got %v, want %v", event.CreationTime, createdAt)
		}

		if event.Preview != expectedPreview {
			t.Errorf("unexpected event preview: got %q, want %q", event.Preview, expectedPreview)
		}
	})
}

func TestChecker_GitHubPullRequest_SendsFormattedUpdate(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)

		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestBaseSubscriptionService(t, db)

		const chatID int64 = 202
		const trackedURL = "https://github.com/user/repo"

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		_, err = subscriptionService.AddLink(ctx, chatID, trackedURL, []string{"backend"})
		if err != nil {
			t.Fatalf("add link: %v", err)
		}

		lastUpdated := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
		if err != nil {
			t.Fatalf("update last updated: %v", err)
		}

		createdAt := time.Date(2026, 4, 3, 9, 45, 0, 0, time.UTC)
		longBody := strings.Repeat("pull request description ", 30)

		githubServer := newGitHubServer(t, []map[string]any{
			{
				"title":      "Add batch processing",
				"body":       longBody,
				"created_at": createdAt.Format(time.RFC3339),
				"user": map[string]any{
					"login": "alice",
				},
				"pull_request": map[string]any{},
			},
		})
		defer githubServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		githubClient := githubhttp.NewClient(githubServer.URL, httpClient, testRetryConfig(), testCircuitBreakerConfig())
		rawSender := newRawUpdateSenderStub()

		checker := appupdates.NewChecker(
			logger,
			100,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			rawSender,
			[]appupdates.LinkClient{
				appupdates.NewGitHubClient(githubClient),
			},
		)

		expectedPreview := normalizeAndTrimPreview(longBody, 200)

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Fatalf("checker returned error: %v", err)
		}

		if len(rawSender.problems) != 0 {
			t.Errorf("unexpected problem batches count: got %d, want 0", len(rawSender.problems))
		}

		if len(rawSender.updates) != 1 {
			t.Fatalf("unexpected sent raw updates count: got %d, want 1", len(rawSender.updates))
		}

		updateMsg := rawSender.updates[0]

		if updateMsg.URL != trackedURL {
			t.Errorf("unexpected update url: got %q, want %q", updateMsg.URL, trackedURL)
		}

		slices.Sort(updateMsg.TgChatIDs)
		if !slices.Equal(updateMsg.TgChatIDs, []int64{chatID}) {
			t.Errorf("unexpected chat ids: got %v, want [%d]", updateMsg.TgChatIDs, chatID)
		}

		if len(updateMsg.Events) != 1 {
			t.Fatalf("unexpected events count: got %d, want 1", len(updateMsg.Events))
		}

		event := updateMsg.Events[0]

		if event.Source != string(schedulerlink.TypeGitHub) {
			t.Errorf("unexpected event source: got %q, want %q", event.Source, string(schedulerlink.TypeGitHub))
		}

		if event.Type != "pull_request" {
			t.Errorf("unexpected event type: got %q, want %q", event.Type, "pull_request")
		}

		if event.Title != "Add batch processing" {
			t.Errorf("unexpected event title: got %q, want %q", event.Title, "Add batch processing")
		}

		if event.Author != "alice" {
			t.Errorf("unexpected event author: got %q, want %q", event.Author, "alice")
		}

		if !event.CreationTime.Equal(createdAt) {
			t.Errorf("unexpected event creation time: got %v, want %v", event.CreationTime, createdAt)
		}

		if event.Preview != expectedPreview {
			t.Errorf("unexpected event preview: got %q, want %q", event.Preview, expectedPreview)
		}
	})
}

func newGitHubServer(t *testing.T, response any) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected github method: got %s, want %s", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/repos/user/repo/issues" {
			t.Errorf("unexpected github path: got %s, want %s", r.URL.Path, "/repos/user/repo/issues")
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

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode github response: %v", err)
		}
	}))
}

func newBotServer(t *testing.T) (*[]bothttp.LinkUpdate, *httptest.Server) {
	t.Helper()

	var mu sync.Mutex
	updates := make([]bothttp.LinkUpdate, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected bot method: got %s, want %s", r.Method, http.MethodPost)
		}

		if r.URL.Path != "/updates" {
			t.Errorf("unexpected bot path: got %s, want %s", r.URL.Path, "/updates")
		}

		var update bothttp.LinkUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			t.Fatalf("decode bot update: %v", err)
		}

		mu.Lock()
		updates = append(updates, update)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))

	return &updates, server
}

func normalizeAndTrimPreview(text string, limit int) string {
	normalized := strings.Join(strings.Fields(text), " ")
	if len(normalized) <= limit {
		return normalized
	}

	return strings.TrimSpace(normalized[:limit])
}
