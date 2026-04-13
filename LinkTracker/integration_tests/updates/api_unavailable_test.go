package updatestest

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	appupdates "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
	stackoverflowhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/stackoverflow"
	httpsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
)

func TestChecker_GitHubUnavailable_DoesNotCrashAndSendsProblemMessage(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)
		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestSubscriptionService(t, db)

		const chatID int64 = 401
		const trackedURL = "https://github.com/user/repo"

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		_, err = subscriptionService.AddLink(ctx, chatID, trackedURL, []string{"backend"})
		if err != nil {
			t.Fatalf("add link: %v", err)
		}

		err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("update last updated: %v", err)
		}

		githubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
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

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Errorf("checker must not crash on github unavailability: %v", err)
		}

		if len(*receivedUpdates) != 1 {
			t.Fatalf("unexpected sent messages count: got %d, want 1", len(*receivedUpdates))
		}

		msg := (*receivedUpdates)[0]

		if len(msg.TgChatIDs) != 1 || msg.TgChatIDs[0] != chatID {
			t.Errorf("unexpected chat ids in problem message: got %v, want [%d]", msg.TgChatIDs, chatID)
		}

		if !strings.Contains(msg.Description, trackedURL) {
			t.Errorf("problem message must mention tracked url, got %q", msg.Description)
		}

		if !strings.Contains(strings.ToLower(msg.Description), "unexpected status") {
			t.Errorf("problem message must mention request failure reason, got %q", msg.Description)
		}
	})
}

func TestChecker_StackOverflowUnavailable_DoesNotCrashAndSendsProblemMessage(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)
		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestSubscriptionService(t, db)

		const chatID int64 = 402
		const trackedURL = "https://stackoverflow.com/questions/123/test"

		err := subscriptionService.RegisterChat(ctx, chatID)
		if err != nil {
			t.Fatalf("register chat: %v", err)
		}

		_, err = subscriptionService.AddLink(ctx, chatID, trackedURL, []string{"qa"})
		if err != nil {
			t.Fatalf("add link: %v", err)
		}

		err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("update last updated: %v", err)
		}

		stackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
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

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Errorf("checker must not crash on stackoverflow unavailability: %v", err)
		}

		if len(*receivedUpdates) != 1 {
			t.Fatalf("unexpected sent messages count: got %d, want 1", len(*receivedUpdates))
		}

		msg := (*receivedUpdates)[0]

		if len(msg.TgChatIDs) != 1 || msg.TgChatIDs[0] != chatID {
			t.Errorf("unexpected chat ids in problem message: got %v, want [%d]", msg.TgChatIDs, chatID)
		}

		if !strings.Contains(msg.Description, trackedURL) {
			t.Errorf("problem message must mention tracked url, got %q", msg.Description)
		}

		if !strings.Contains(strings.ToLower(msg.Description), "unexpected status") {
			t.Errorf("problem message must mention request failure reason, got %q", msg.Description)
		}
	})
}
