package updates

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/memory"
)

type fakeLinkClient struct {
	linkType     schedulerlink.LinkType
	getUpdatedAt func(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error)
}

func (c fakeLinkClient) Type() schedulerlink.LinkType {
	return c.linkType
}

func (c fakeLinkClient) GetUpdatedAt(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
	return c.getUpdatedAt(ctx, link)
}

func TestChecker_Check_SendsUpdatesOnlyToSubscribedChats(t *testing.T) {
	const trackedURL = "https://github.com/user/repo"
	const otherURL = "https://github.com/other/repo"

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()

	err := subscriptionService.RegisterChat(1)
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.RegisterChat(2)
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.RegisterChat(3)
	if err != nil {
		t.Fatal(err)
	}

	_, err = subscriptionService.AddLink(1, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = subscriptionService.AddLink(2, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = subscriptionService.AddLink(3, otherURL, []string{"other"})
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.UpdateLastUpdated(trackedURL, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.UpdateLastUpdated(otherURL, time.Unix(200, 0))
	if err != nil {
		t.Fatal(err)
	}

	gotUpdates, botServer := newTestBotServer(t)
	defer botServer.Close()

	githubClient := fakeLinkClient{
		linkType: schedulerlink.TypeGitHub,
		getUpdatedAt: func(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
			githubLink, ok := link.(schedulerlink.GitHubLink)
			if !ok {
				return time.Time{}, errors.New("unexpected link type")
			}

			switch {
			case githubLink.Owner == "user" && githubLink.Repo == "repo":
				return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), nil
			case githubLink.Owner == "other" && githubLink.Repo == "repo":
				return time.Unix(200, 0), nil
			default:
				return time.Time{}, errors.New("unexpected github link")
			}
		},
	}

	checker := NewChecker(
		logger,
		subscriptionService,
		parser,
		bothttp.NewClient(botServer.URL, botServer.Client()),
		githubClient,
	)

	err = checker.Check(context.Background())

	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(*gotUpdates) != 1 {
		t.Errorf("unexpected number of sent updates: got %d, want %d", len(*gotUpdates), 1)
	}

	if len(*gotUpdates) == 1 {
		update := (*gotUpdates)[0]

		if update.URL != trackedURL {
			t.Errorf("unexpected update url: got %q, want %q", update.URL, trackedURL)
		}

		if update.Description != "Link was updated" {
			t.Errorf("unexpected update description: got %q, want %q", update.Description, "Link was updated")
		}

		slices.Sort(update.TgChatIDs)
		wantChatIDs := []int64{1, 2}
		if !slices.Equal(update.TgChatIDs, wantChatIDs) {
			t.Errorf("unexpected update chat ids: got %#v, want %#v", update.TgChatIDs, wantChatIDs)
		}
	}

	trackedURLs, err := subscriptionService.ListTrackedURLs()
	if err != nil {
		t.Fatal(err)
	}

	gotTrackedUpdatedAt := trackedURLs[trackedURL]
	wantTrackedUpdatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !gotTrackedUpdatedAt.Equal(wantTrackedUpdatedAt) {
		t.Errorf("unexpected tracked url updated_at: got %v, want %v", gotTrackedUpdatedAt, wantTrackedUpdatedAt)
	}

	gotOtherUpdatedAt := trackedURLs[otherURL]
	wantOtherUpdatedAt := time.Unix(200, 0)
	if !gotOtherUpdatedAt.Equal(wantOtherUpdatedAt) {
		t.Errorf("unexpected other url updated_at: got %v, want %v", gotOtherUpdatedAt, wantOtherUpdatedAt)
	}
}

func TestChecker_Check_GitHubNon2xxDoesNotCrash(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()

	err := subscriptionService.RegisterChat(1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://github.com/user/repo"

	_, err = subscriptionService.AddLink(1, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.UpdateLastUpdated(trackedURL, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}

	gotUpdates, botServer := newTestBotServer(t)
	defer botServer.Close()

	githubClient := fakeLinkClient{
		linkType: schedulerlink.TypeGitHub,
		getUpdatedAt: func(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
			return time.Time{}, errors.New("github returned unexpected status: 500 Internal Server Error")
		},
	}

	checker := NewChecker(
		logger,
		subscriptionService,
		parser,
		bothttp.NewClient(botServer.URL, botServer.Client()),
		githubClient,
	)

	err = checker.Check(context.Background())

	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(*gotUpdates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(*gotUpdates), 0)
	}
}

func TestChecker_Check_GitHubInvalidBodyDoesNotCrash(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()

	err := subscriptionService.RegisterChat(1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://github.com/user/repo"

	_, err = subscriptionService.AddLink(1, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.UpdateLastUpdated(trackedURL, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}

	gotUpdates, botServer := newTestBotServer(t)
	defer botServer.Close()

	githubClient := fakeLinkClient{
		linkType: schedulerlink.TypeGitHub,
		getUpdatedAt: func(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
			return time.Time{}, errors.New("decode github response: json: cannot unmarshal number into Go struct field RepositoryResponse.updated_at of type time.Time")
		},
	}

	checker := NewChecker(
		logger,
		subscriptionService,
		parser,
		bothttp.NewClient(botServer.URL, botServer.Client()),
		githubClient,
	)

	err = checker.Check(context.Background())

	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(*gotUpdates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(*gotUpdates), 0)
	}
}

func TestChecker_Check_StackOverflowNon2xxDoesNotCrash(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()

	err := subscriptionService.RegisterChat(1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://stackoverflow.com/questions/123/test"

	_, err = subscriptionService.AddLink(1, trackedURL, []string{"qa"})
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.UpdateLastUpdated(trackedURL, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}

	gotUpdates, botServer := newTestBotServer(t)
	defer botServer.Close()

	stackClient := fakeLinkClient{
		linkType: schedulerlink.TypeStackOverflow,
		getUpdatedAt: func(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
			return time.Time{}, errors.New("stackoverflow returned unexpected status: 502 Bad Gateway")
		},
	}

	checker := NewChecker(
		logger,
		subscriptionService,
		parser,
		bothttp.NewClient(botServer.URL, botServer.Client()),
		stackClient,
	)

	err = checker.Check(context.Background())

	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(*gotUpdates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(*gotUpdates), 0)
	}
}

func TestChecker_Check_StackOverflowInvalidBodyDoesNotCrash(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()

	err := subscriptionService.RegisterChat(1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://stackoverflow.com/questions/123/test"

	_, err = subscriptionService.AddLink(1, trackedURL, []string{"qa"})
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.UpdateLastUpdated(trackedURL, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}

	gotUpdates, botServer := newTestBotServer(t)
	defer botServer.Close()

	stackClient := fakeLinkClient{
		linkType: schedulerlink.TypeStackOverflow,
		getUpdatedAt: func(ctx context.Context, link schedulerlink.SchedulerLink) (time.Time, error) {
			return time.Time{}, errors.New("decode stackoverflow response: json: cannot unmarshal string into Go struct field QuestionResponse.items of type []stackoverflow.Question")
		},
	}

	checker := NewChecker(
		logger,
		subscriptionService,
		parser,
		bothttp.NewClient(botServer.URL, botServer.Client()),
		stackClient,
	)

	err = checker.Check(context.Background())

	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(*gotUpdates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(*gotUpdates), 0)
	}
}

func newTestSubscriptionService() *service.SubscriptionService {
	chatRepository := memory.NewChatRepository()
	subscriptionRepository := memory.NewSubscriptionRepository()
	return service.NewSubscriptionService(chatRepository, subscriptionRepository)
}

func newTestBotServer(t *testing.T) (*[]bothttp.LinkUpdate, *httptest.Server) {
	t.Helper()

	gotUpdates := make([]bothttp.LinkUpdate, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected bot method: got %s, want %s", r.Method, http.MethodPost)
		}

		if r.URL.Path != "/updates" {
			t.Errorf("unexpected bot path: got %s, want %s", r.URL.Path, "/updates")
		}

		var update bothttp.LinkUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			t.Errorf("failed to decode bot update: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		gotUpdates = append(gotUpdates, update)
		w.WriteHeader(http.StatusOK)
	}))

	return &gotUpdates, server
}
