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
	appupdates "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
	stackoverflowhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/stackoverflow"
	httpsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
)

func TestChecker_StackOverflowAnswer_SendsFormattedUpdate(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)
		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestSubscriptionService(t, db)

		const chatID int64 = 301
		const trackedURL = "https://stackoverflow.com/questions/123/test"

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		_, err = subscriptionService.AddLink(ctx, chatID, trackedURL, []string{"qa"})
		if err != nil {
			t.Fatalf("add link: %v", err)
		}

		lastUpdated := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
		if err != nil {
			t.Fatalf("update last updated: %v", err)
		}

		createdAt := time.Date(2026, 4, 2, 12, 30, 0, 0, time.UTC)
		longBody := "<p>" + strings.Repeat("answer body ", 40) + "</p>"

		stackServer := newStackOverflowServer(
			t,
			map[string]any{
				"items": []map[string]any{
					{
						"title": "How to use context in Go?",
					},
				},
			},
			map[string]any{
				"items": []map[string]any{
					{
						"creation_date": createdAt.Unix(),
						"body":          longBody,
						"owner": map[string]any{
							"display_name": "alice",
						},
					},
				},
			},
			map[string]any{
				"items": []map[string]any{},
			},
		)
		defer stackServer.Close()

		receivedUpdates, botServer := newBotServer(t)
		defer botServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		stackClient := stackoverflowhttp.NewClient(stackServer.URL, httpClient)
		botClient := bothttp.NewClient(botServer.URL, httpClient)

		checker := appupdates.NewChecker(
			logger,
			100,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			httpsender.NewHTTPSender(botClient),
			[]appupdates.LinkClient{
				appupdates.NewStackOverflowClient(stackClient),
			},
			[]appupdates.Formatter{
				appupdates.StackOverflowFormatter{},
			},
		)

		expectedPreview := normalizeAndTrimStackPreview(longBody, 200)

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

		if !strings.Contains(update.Description, "ответ") {
			t.Errorf("update description must contain answer label, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "How to use context in Go?") {
			t.Errorf("update description must contain question title, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "alice") {
			t.Errorf("update description must contain username, got %q", update.Description)
		}

		expectedTime := createdAt.Format("02 Jan 2006 15:04")
		if !strings.Contains(update.Description, expectedTime) {
			t.Errorf("update description must contain formatted creation time, got %q", update.Description)
		}

		if !strings.Contains(update.Description, expectedPreview) {
			t.Errorf("update description must contain trimmed preview, got %q", update.Description)
		}
	})
}

func TestChecker_StackOverflowComment_SendsFormattedUpdate(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)
		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestSubscriptionService(t, db)

		const chatID int64 = 302
		const trackedURL = "https://stackoverflow.com/questions/123/test"

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		_, err = subscriptionService.AddLink(ctx, chatID, trackedURL, []string{"qa"})
		if err != nil {
			t.Fatalf("add link: %v", err)
		}

		lastUpdated := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
		if err != nil {
			t.Fatalf("update last updated: %v", err)
		}

		createdAt := time.Date(2026, 4, 3, 9, 45, 0, 0, time.UTC)
		longBody := "<div>" + strings.Repeat("comment body ", 40) + "</div>"

		stackServer := newStackOverflowServer(
			t,
			map[string]any{
				"items": []map[string]any{
					{
						"title": "What is an interface in Go?",
					},
				},
			},
			map[string]any{
				"items": []map[string]any{},
			},
			map[string]any{
				"items": []map[string]any{
					{
						"creation_date": createdAt.Unix(),
						"body":          longBody,
						"owner": map[string]any{
							"display_name": "bob",
						},
					},
				},
			},
		)
		defer stackServer.Close()

		receivedUpdates, botServer := newBotServer(t)
		defer botServer.Close()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		stackClient := stackoverflowhttp.NewClient(stackServer.URL, httpClient)
		botClient := bothttp.NewClient(botServer.URL, httpClient)

		checker := appupdates.NewChecker(
			logger,
			100,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			httpsender.NewHTTPSender(botClient),
			[]appupdates.LinkClient{
				appupdates.NewStackOverflowClient(stackClient),
			},
			[]appupdates.Formatter{
				appupdates.StackOverflowFormatter{},
			},
		)

		expectedPreview := normalizeAndTrimStackPreview(longBody, 200)

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

		if !strings.Contains(update.Description, "комментарий") {
			t.Errorf("update description must contain comment label, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "What is an interface in Go?") {
			t.Errorf("update description must contain question title, got %q", update.Description)
		}

		if !strings.Contains(update.Description, "bob") {
			t.Errorf("update description must contain username, got %q", update.Description)
		}

		expectedTime := createdAt.Format("02 Jan 2006 15:04")
		if !strings.Contains(update.Description, expectedTime) {
			t.Errorf("update description must contain formatted creation time, got %q", update.Description)
		}

		if !strings.Contains(update.Description, expectedPreview) {
			t.Errorf("update description must contain trimmed preview, got %q", update.Description)
		}
	})
}

func newStackOverflowServer(
	t *testing.T,
	questionResponse any,
	answersResponse any,
	commentsResponse any,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected stackoverflow method: got %s, want %s", r.Method, http.MethodGet)
		}

		switch r.URL.Path {
		case "/questions/123":
			if got := r.URL.Query().Get("site"); got != "stackoverflow" {
				t.Errorf("unexpected question query site: got %q, want %q", got, "stackoverflow")
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(questionResponse); err != nil {
				t.Fatalf("encode question response: %v", err)
			}

		case "/questions/123/answers":
			if got := r.URL.Query().Get("site"); got != "stackoverflow" {
				t.Errorf("unexpected answers query site: got %q, want %q", got, "stackoverflow")
			}
			if got := r.URL.Query().Get("sort"); got != "creation" {
				t.Errorf("unexpected answers query sort: got %q, want %q", got, "creation")
			}
			if got := r.URL.Query().Get("order"); got != "desc" {
				t.Errorf("unexpected answers query order: got %q, want %q", got, "desc")
			}
			if got := r.URL.Query().Get("filter"); got != "withbody" {
				t.Errorf("unexpected answers query filter: got %q, want %q", got, "withbody")
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(answersResponse); err != nil {
				t.Fatalf("encode answers response: %v", err)
			}

		case "/questions/123/comments":
			if got := r.URL.Query().Get("site"); got != "stackoverflow" {
				t.Errorf("unexpected comments query site: got %q, want %q", got, "stackoverflow")
			}
			if got := r.URL.Query().Get("sort"); got != "creation" {
				t.Errorf("unexpected comments query sort: got %q, want %q", got, "creation")
			}
			if got := r.URL.Query().Get("order"); got != "desc" {
				t.Errorf("unexpected comments query order: got %q, want %q", got, "desc")
			}
			if got := r.URL.Query().Get("filter"); got != "withbody" {
				t.Errorf("unexpected comments query filter: got %q, want %q", got, "withbody")
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(commentsResponse); err != nil {
				t.Fatalf("encode comments response: %v", err)
			}

		default:
			t.Errorf("unexpected stackoverflow path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func normalizeAndTrimStackPreview(text string, limit int) string {
	replacer := strings.NewReplacer(
		"<p>", " ", "</p>", " ",
		"<div>", " ", "</div>", " ",
	)
	normalized := replacer.Replace(text)
	normalized = strings.Join(strings.Fields(normalized), " ")

	if len(normalized) <= limit {
		return normalized
	}

	return strings.TrimSpace(normalized[:limit])
}
