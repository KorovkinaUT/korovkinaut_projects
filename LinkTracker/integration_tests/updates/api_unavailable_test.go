package updatestest

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	appupdates "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
	stackoverflowhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/stackoverflow"
)

func TestChecker_GitHubUnavailable_DoesNotCrashAndSendsProblemMessage(t *testing.T) {
	helpers.RunAllAccessTypes(t, func(t *testing.T, accessType string) {
		// arrange
		ctx := context.Background()

		db := helpers.NewTestDatabase(t, accessType)
		defer db.Close(t)
		helpers.ApplyMigrations(t, db)

		subscriptionService := helpers.NewTestBaseSubscriptionService(t, db)

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

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Errorf("checker must not crash on github unavailability: %v", err)
		}

		if len(rawSender.updates) != 0 {
			t.Errorf("unexpected raw updates count: got %d, want 0", len(rawSender.updates))
		}

		if len(rawSender.problems) != 1 {
			t.Fatalf("unexpected problem batches count: got %d, want 1", len(rawSender.problems))
		}

		problems := rawSender.problems[0]
		if len(problems) != 1 {
			t.Fatalf("unexpected problems count: got %d, want 1", len(problems))
		}

		problem := problems[0]

		if problem.URL != trackedURL {
			t.Errorf("unexpected problem url: got %q, want %q", problem.URL, trackedURL)
		}

		if len(problem.ChatIDs) != 1 || problem.ChatIDs[0] != chatID {
			t.Errorf("unexpected chat ids in problem: got %v, want [%d]", problem.ChatIDs, chatID)
		}

		if !strings.Contains(strings.ToLower(problem.Message), "unexpected status") {
			t.Errorf("problem must mention request failure reason, got %q", problem.Message)
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

		subscriptionService := helpers.NewTestBaseSubscriptionService(t, db)

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

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		httpClient := &http.Client{Timeout: 5 * time.Second}

		stackClient := stackoverflowhttp.NewClient(stackServer.URL, httpClient, testRetryConfig(), testCircuitBreakerConfig())
		rawSender := newRawUpdateSenderStub()

		checker := appupdates.NewChecker(
			logger,
			100,
			1,
			subscriptionService,
			schedulerlink.NewService(),
			rawSender,
			[]appupdates.LinkClient{
				appupdates.NewStackOverflowClient(stackClient),
			},
		)

		// act
		err = checker.Check(ctx)

		// assert
		if err != nil {
			t.Errorf("checker must not crash on stackoverflow unavailability: %v", err)
		}

		if len(rawSender.updates) != 0 {
			t.Errorf("unexpected raw updates count: got %d, want 0", len(rawSender.updates))
		}

		if len(rawSender.problems) != 1 {
			t.Fatalf("unexpected problem batches count: got %d, want 1", len(rawSender.problems))
		}

		problems := rawSender.problems[0]
		if len(problems) != 1 {
			t.Fatalf("unexpected problems count: got %d, want 1", len(problems))
		}

		problem := problems[0]

		if problem.URL != trackedURL {
			t.Errorf("unexpected problem url: got %q, want %q", problem.URL, trackedURL)
		}

		if len(problem.ChatIDs) != 1 || problem.ChatIDs[0] != chatID {
			t.Errorf("unexpected chat ids in problem: got %v, want [%d]", problem.ChatIDs, chatID)
		}

		if !strings.Contains(strings.ToLower(problem.Message), "unexpected status") {
			t.Errorf("problem must mention request failure reason, got %q", problem.Message)
		}
	})
}

func testRetryConfig() *config.RetryConfig {
	return &config.RetryConfig{
		Attempts:          1,
		Delay:             time.Millisecond,
		RetryableStatuses: []int{},
	}
}

func testCircuitBreakerConfig() *config.CircuitBreakerConfig {
	return &config.CircuitBreakerConfig{
		SlidingWindowSize:       1000,
		FailureRateThreshold:    100,
		CallsInHalfOpen:         1,
		WaitDurationInOpenState: time.Millisecond,
	}
}

type rawUpdateSenderStub struct {
	mu       sync.Mutex
	updates  []sender.RawUpdateEvents
	problems [][]sender.Problem
}

func newRawUpdateSenderStub() *rawUpdateSenderStub {
	return &rawUpdateSenderStub{
		updates:  make([]sender.RawUpdateEvents, 0),
		problems: make([][]sender.Problem, 0),
	}
}

func (s *rawUpdateSenderStub) SendUpdateEvents(ctx context.Context, msg sender.RawUpdateEvents) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.updates = append(s.updates, msg)
	return nil
}

func (s *rawUpdateSenderStub) SendProblems(ctx context.Context, msg []sender.Problem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.problems = append(s.problems, msg)
	return nil
}
