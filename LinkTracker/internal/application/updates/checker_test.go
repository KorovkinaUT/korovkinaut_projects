package updates

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/memory"
)

type fakeLinkClient struct {
	linkType     schedulerlink.LinkType
	getNewEvents func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error)
}

func (c fakeLinkClient) Type() schedulerlink.LinkType {
	return c.linkType
}

func (c fakeLinkClient) GetNewEvents(
	ctx context.Context,
	link schedulerlink.SchedulerLink,
	since time.Time,
) ([]update.Event, error) {
	return c.getNewEvents(ctx, link, since)
}

type fakeFormatter struct {
	linkType schedulerlink.LinkType
	format   func(rawURL string, events []update.Event) (string, error)
}

func (f fakeFormatter) Type() schedulerlink.LinkType {
	return f.linkType
}

func (f fakeFormatter) Format(rawURL string, events []update.Event) (string, error) {
	return f.format(rawURL, events)
}

type fakeMessageSender struct {
	mu       sync.Mutex
	updates  []sender.UpdateMessage
	problems []sender.ProblemsMessage
}

func (s *fakeMessageSender) SendUpdate(ctx context.Context, msg sender.UpdateMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.updates = append(s.updates, msg)
	return nil
}

func (s *fakeMessageSender) SendProblems(ctx context.Context, msg sender.ProblemsMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.problems = append(s.problems, msg)
	return nil
}

func TestChecker_Check_SendsUpdatesOnlyToSubscribedChats(t *testing.T) {
	// arrange
	ctx := context.Background()

	const trackedURL = "https://github.com/user/repo"
	const otherURL = "https://github.com/other/repo"

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeMessageSender{}

	err := subscriptionService.RegisterChat(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.RegisterChat(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	err = subscriptionService.RegisterChat(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}

	_, err = subscriptionService.AddLink(ctx, 1, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = subscriptionService.AddLink(ctx, 2, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = subscriptionService.AddLink(ctx, 3, otherURL, []string{"other"})
	if err != nil {
		t.Fatal(err)
	}

	trackedLastUpdated := time.Unix(100, 0).UTC()
	err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, trackedLastUpdated)
	if err != nil {
		t.Fatal(err)
	}

	otherLastUpdated := time.Unix(200, 0).UTC()
	err = subscriptionService.UpdateLastUpdated(ctx, otherURL, otherLastUpdated)
	if err != nil {
		t.Fatal(err)
	}

	githubClient := fakeLinkClient{
		linkType: schedulerlink.TypeGitHub,
		getNewEvents: func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error) {
			githubLink, ok := link.(schedulerlink.GitHubLink)
			if !ok {
				return nil, errors.New("unexpected link type")
			}

			switch {
			case githubLink.Owner == "user" && githubLink.Repo == "repo":
				if !since.Equal(trackedLastUpdated) {
					return nil, errors.New("unexpected since for tracked url")
				}

				return []update.Event{
					update.GitHubEvent{
						Type:         update.GitHubEventIssue,
						Title:        "Issue title",
						Username:     "alice",
						CreationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
						Preview:      "Issue preview",
					},
				}, nil
			case githubLink.Owner == "other" && githubLink.Repo == "repo":
				if !since.Equal(otherLastUpdated) {
					return nil, errors.New("unexpected since for other url")
				}

				return nil, nil
			default:
				return nil, errors.New("unexpected github link")
			}
		},
	}

	githubFormatter := fakeFormatter{
		linkType: schedulerlink.TypeGitHub,
		format: func(rawURL string, events []update.Event) (string, error) {
			return "Link was updated", nil
		},
	}

	checker := NewChecker(
		logger,
		100,
		1,
		subscriptionService,
		parser,
		messageSender,
		[]LinkClient{githubClient},
		[]Formatter{githubFormatter},
	)

	// act
	err = checker.Check(ctx)

	// assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 1 {
		t.Errorf("unexpected number of sent updates: got %d, want %d", len(messageSender.updates), 1)
	}

	if len(messageSender.problems) != 0 {
		t.Errorf("unexpected number of problem messages: got %d, want %d", len(messageSender.problems), 0)
	}

	if len(messageSender.updates) == 1 {
		gotUpdate := messageSender.updates[0]

		if gotUpdate.URL != trackedURL {
			t.Errorf("unexpected update url: got %q, want %q", gotUpdate.URL, trackedURL)
		}

		if gotUpdate.Description != "Link was updated" {
			t.Errorf("unexpected update description: got %q, want %q", gotUpdate.Description, "Link was updated")
		}

		slices.Sort(gotUpdate.TgChatIDs)
		wantChatIDs := []int64{1, 2}
		if !slices.Equal(gotUpdate.TgChatIDs, wantChatIDs) {
			t.Errorf("unexpected update chat ids: got %#v, want %#v", gotUpdate.TgChatIDs, wantChatIDs)
		}
	}

	trackedURLs, err := subscriptionService.ListTrackedURLsAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	gotTrackedUpdatedAt := trackedURLs[trackedURL]
	wantTrackedUpdatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if !gotTrackedUpdatedAt.Equal(wantTrackedUpdatedAt) {
		t.Errorf("unexpected tracked url updated_at: got %v, want %v", gotTrackedUpdatedAt, wantTrackedUpdatedAt)
	}

	gotOtherUpdatedAt := trackedURLs[otherURL]
	wantOtherUpdatedAt := otherLastUpdated
	if !gotOtherUpdatedAt.Equal(wantOtherUpdatedAt) {
		t.Errorf("unexpected other url updated_at: got %v, want %v", gotOtherUpdatedAt, wantOtherUpdatedAt)
	}
}

func TestChecker_Check_GitHubNon2xxDoesNotCrash(t *testing.T) {
	// arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeMessageSender{}

	err := subscriptionService.RegisterChat(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://github.com/user/repo"

	_, err = subscriptionService.AddLink(ctx, 1, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	lastUpdated := time.Unix(100, 0).UTC()
	err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
	if err != nil {
		t.Fatal(err)
	}

	githubClient := fakeLinkClient{
		linkType: schedulerlink.TypeGitHub,
		getNewEvents: func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error) {
			if !since.Equal(lastUpdated) {
				return nil, errors.New("unexpected since")
			}

			return nil, errors.New("github returned unexpected status: 500 Internal Server Error")
		},
	}

	checker := NewChecker(
		logger,
		100,
		1,
		subscriptionService,
		parser,
		messageSender,
		[]LinkClient{githubClient},
		nil,
	)

	// act
	err = checker.Check(ctx)

	// assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem messages: got %d, want %d", len(messageSender.problems), 1)
	}
}

func TestChecker_Check_GitHubInvalidBodyDoesNotCrash(t *testing.T) {
	// arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeMessageSender{}

	err := subscriptionService.RegisterChat(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://github.com/user/repo"

	_, err = subscriptionService.AddLink(ctx, 1, trackedURL, []string{"backend"})
	if err != nil {
		t.Fatal(err)
	}

	lastUpdated := time.Unix(100, 0).UTC()
	err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
	if err != nil {
		t.Fatal(err)
	}

	githubClient := fakeLinkClient{
		linkType: schedulerlink.TypeGitHub,
		getNewEvents: func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error) {
			if !since.Equal(lastUpdated) {
				return nil, errors.New("unexpected since")
			}

			return nil, errors.New("decode github response: json: cannot unmarshal number into Go struct field RepositoryResponse.updated_at of type time.Time")
		},
	}

	checker := NewChecker(
		logger,
		100,
		1,
		subscriptionService,
		parser,
		messageSender,
		[]LinkClient{githubClient},
		nil,
	)

	// act
	err = checker.Check(ctx)

	// assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem messages: got %d, want %d", len(messageSender.problems), 1)
	}
}

func TestChecker_Check_StackOverflowNon2xxDoesNotCrash(t *testing.T) {
	// arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeMessageSender{}

	err := subscriptionService.RegisterChat(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://stackoverflow.com/questions/123/test"

	_, err = subscriptionService.AddLink(ctx, 1, trackedURL, []string{"qa"})
	if err != nil {
		t.Fatal(err)
	}

	lastUpdated := time.Unix(100, 0).UTC()
	err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
	if err != nil {
		t.Fatal(err)
	}

	stackClient := fakeLinkClient{
		linkType: schedulerlink.TypeStackOverflow,
		getNewEvents: func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error) {
			if !since.Equal(lastUpdated) {
				return nil, errors.New("unexpected since")
			}

			return nil, errors.New("stackoverflow returned unexpected status: 502 Bad Gateway")
		},
	}

	checker := NewChecker(
		logger,
		100,
		1,
		subscriptionService,
		parser,
		messageSender,
		[]LinkClient{stackClient},
		nil,
	)

	// act
	err = checker.Check(ctx)

	// assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem messages: got %d, want %d", len(messageSender.problems), 1)
	}
}

func TestChecker_Check_StackOverflowInvalidBodyDoesNotCrash(t *testing.T) {
	// arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeMessageSender{}

	err := subscriptionService.RegisterChat(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	const trackedURL = "https://stackoverflow.com/questions/123/test"

	_, err = subscriptionService.AddLink(ctx, 1, trackedURL, []string{"qa"})
	if err != nil {
		t.Fatal(err)
	}

	lastUpdated := time.Unix(100, 0).UTC()
	err = subscriptionService.UpdateLastUpdated(ctx, trackedURL, lastUpdated)
	if err != nil {
		t.Fatal(err)
	}

	stackClient := fakeLinkClient{
		linkType: schedulerlink.TypeStackOverflow,
		getNewEvents: func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error) {
			if !since.Equal(lastUpdated) {
				return nil, errors.New("unexpected since")
			}

			return nil, errors.New("decode stackoverflow response: json: cannot unmarshal string into Go struct field QuestionResponse.items of type []stackoverflow.Question")
		},
	}

	checker := NewChecker(
		logger,
		100,
		1,
		subscriptionService,
		parser,
		messageSender,
		[]LinkClient{stackClient},
		nil,
	)

	// act
	err = checker.Check(ctx)

	// assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem messages: got %d, want %d", len(messageSender.problems), 1)
	}
}

func newTestSubscriptionService() *service.SubscriptionService {
	chatRepository := memory.NewChatRepository()
	subscriptionRepository := memory.NewSubscriptionRepository()
	return service.NewSubscriptionService(chatRepository, subscriptionRepository)
}
