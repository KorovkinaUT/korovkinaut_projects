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
	httpsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
)

func TestChecker_GitHubIssue_SendsFormattedUpdate(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)

		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestSubscriptionService(t, db)

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

		receivedUpdates, botServer := newBotServer(t)
		defer botServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		githubClient := githubhttp.NewClient(githubServer.URL, httpClient)
		botClient := bothttp.NewClient(botServer.URL, httpClient)

		checker := appupdates.NewChecker(
			logger,
			100,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			httpsender.NewHTTPSender(botClient),
			[]appupdates.LinkClient{
				appupdates.NewGitHubClient(githubClient),
			},
			[]appupdates.Formatter{
				appupdates.GitHubFormatter{},
			},
		)

		expectedPreview := normalizeAndTrimPreview(longBody, 200)

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Fatalf("checker returned error: %v", err)
		}

		if len(*receivedUpdates) != 1 {
			t.Fatalf("unexpected sent updates count: got %d, want 1", len(*receivedUpdates))
		}

		update := (*receivedUpdates)[0]

		if update.URL != trackedURL {
			t.Errorf("unexpected update url: got %q, want %q", update.URL, trackedURL)
		}

		slices.Sort(update.TgChatIDs)
		if !slices.Equal(update.TgChatIDs, []int64{chatID}) {
			t.Errorf("unexpected chat ids: got %v, want [%d]", update.TgChatIDs, chatID)
		}

		if !strings.Contains(update.Description, trackedURL) {
			t.Errorf("update description must mention tracked url, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "Issue") {
			t.Errorf("update description must contain issue label, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "Fix race in checker") {
			t.Errorf("update description must contain issue title, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "octocat") {
			t.Errorf("update description must contain username, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "02 Apr 2026 11:30") {
			t.Errorf("update description must contain formatted creation time, got %q", update.Description)
		}

		if !strings.Contains(update.Description, expectedPreview) {
			t.Errorf("update description must contain trimmed preview, got %q", update.Description)
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

		subscriptionService := helpers.NewTestSubscriptionService(t, db)

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

		receivedUpdates, botServer := newBotServer(t)
		defer botServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		githubClient := githubhttp.NewClient(githubServer.URL, httpClient)
		botClient := bothttp.NewClient(botServer.URL, httpClient)

		checker := appupdates.NewChecker(
			logger,
			100,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			httpsender.NewHTTPSender(botClient),
			[]appupdates.LinkClient{
				appupdates.NewGitHubClient(githubClient),
			},
			[]appupdates.Formatter{
				appupdates.GitHubFormatter{},
			},
		)

		expectedPreview := normalizeAndTrimPreview(longBody, 200)

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Fatalf("checker returned error: %v", err)
		}

		if len(*receivedUpdates) != 1 {
			t.Fatalf("unexpected sent updates count: got %d, want 1", len(*receivedUpdates))
		}

		update := (*receivedUpdates)[0]

		if update.URL != trackedURL {
			t.Errorf("unexpected update url: got %q, want %q", update.URL, trackedURL)
		}

		slices.Sort(update.TgChatIDs)
		if !slices.Equal(update.TgChatIDs, []int64{chatID}) {
			t.Errorf("unexpected chat ids: got %v, want [%d]", update.TgChatIDs, chatID)
		}

		if !strings.Contains(update.Description, trackedURL) {
			t.Errorf("update description must mention tracked url, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "Pull Request") {
			t.Errorf("update description must contain pull request label, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "Add batch processing") {
			t.Errorf("update description must contain pull request title, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "alice") {
			t.Errorf("update description must contain username, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "03 Apr 2026 09:45") {
			t.Errorf("update description must contain formatted creation time, got %q", update.Description)
		}

		if !strings.Contains(update.Description, expectedPreview) {
			t.Errorf("update description must contain trimmed preview, got %q", update.Description)
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
