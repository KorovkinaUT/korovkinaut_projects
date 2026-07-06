package updates

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
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

type fakeRawUpdateSender struct {
	mu       sync.Mutex
	updates  []sender.RawUpdateEvents
	problems [][]sender.Problem
}

func (s *fakeRawUpdateSender) SendUpdateEvents(ctx context.Context, msg sender.RawUpdateEvents) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.updates = append(s.updates, msg)
	return nil
}

func (s *fakeRawUpdateSender) SendProblems(ctx context.Context, msg []sender.Problem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.problems = append(s.problems, msg)
	return nil
}

func TestChecker_Check_SendsUpdatesOnlyToSubscribedChats(t *testing.T) {
	//arrange
	ctx := context.Background()

	const trackedURL = "https://github.com/user/repo"
	const otherURL = "https://github.com/other/repo"

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeRawUpdateSender{}

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

	eventCreationTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

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
						CreationTime: eventCreationTime,
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

	checker := NewChecker(
		logger,
		100,
		1,
		subscriptionService,
		parser,
		messageSender,
		[]LinkClient{githubClient},
	)

	//act
	err = checker.Check(ctx)

	//assert
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

		slices.Sort(gotUpdate.TgChatIDs)
		wantChatIDs := []int64{1, 2}
		if !slices.Equal(gotUpdate.TgChatIDs, wantChatIDs) {
			t.Errorf("unexpected update chat ids: got %#v, want %#v", gotUpdate.TgChatIDs, wantChatIDs)
		}

		if len(gotUpdate.Events) != 1 {
			t.Errorf("unexpected number of events: got %d, want %d", len(gotUpdate.Events), 1)
		}

		if len(gotUpdate.Events) == 1 {
			gotEvent := gotUpdate.Events[0]

			if gotEvent.Source != string(schedulerlink.TypeGitHub) {
				t.Errorf("unexpected event source: got %q, want %q", gotEvent.Source, string(schedulerlink.TypeGitHub))
			}

			if gotEvent.Type != string(update.GitHubEventIssue) {
				t.Errorf("unexpected event type: got %q, want %q", gotEvent.Type, string(update.GitHubEventIssue))
			}

			if gotEvent.Title != "Issue title" {
				t.Errorf("unexpected event title: got %q, want %q", gotEvent.Title, "Issue title")
			}

			if gotEvent.Author != "alice" {
				t.Errorf("unexpected event author: got %q, want %q", gotEvent.Author, "alice")
			}

			if gotEvent.Preview != "Issue preview" {
				t.Errorf("unexpected event preview: got %q, want %q", gotEvent.Preview, "Issue preview")
			}

			if !gotEvent.CreationTime.Equal(eventCreationTime) {
				t.Errorf("unexpected event creation time: got %v, want %v", gotEvent.CreationTime, eventCreationTime)
			}
		}
	}

	trackedURLs, err := subscriptionService.ListTrackedURLsAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	gotTrackedUpdatedAt := trackedURLs[trackedURL]
	wantTrackedUpdatedAt := eventCreationTime
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
	//arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeRawUpdateSender{}

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
	)

	//act
	err = checker.Check(ctx)

	//assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem batches: got %d, want %d", len(messageSender.problems), 1)
	}

	if len(messageSender.problems) == 1 {
		gotProblems := messageSender.problems[0]

		if len(gotProblems) != 1 {
			t.Errorf("unexpected problems count: got %d, want %d", len(gotProblems), 1)
		}

		if len(gotProblems) == 1 {
			assertProblem(t, gotProblems[0], trackedURL, "github returned unexpected status: 500 Internal Server Error", []int64{1})
		}
	}
}

func TestChecker_Check_GitHubInvalidBodyDoesNotCrash(t *testing.T) {
	//arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeRawUpdateSender{}

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

	expectedProblem := "decode github response: json: cannot unmarshal number into Go struct field RepositoryResponse.updated_at of type time.Time"

	githubClient := fakeLinkClient{
		linkType: schedulerlink.TypeGitHub,
		getNewEvents: func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error) {
			if !since.Equal(lastUpdated) {
				return nil, errors.New("unexpected since")
			}

			return nil, errors.New(expectedProblem)
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
	)

	//act
	err = checker.Check(ctx)

	//assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem batches: got %d, want %d", len(messageSender.problems), 1)
	}

	if len(messageSender.problems) == 1 {
		gotProblems := messageSender.problems[0]

		if len(gotProblems) != 1 {
			t.Errorf("unexpected problems count: got %d, want %d", len(gotProblems), 1)
		}

		if len(gotProblems) == 1 {
			assertProblem(t, gotProblems[0], trackedURL, expectedProblem, []int64{1})
		}
	}
}

func TestChecker_Check_StackOverflowNon2xxDoesNotCrash(t *testing.T) {
	//arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeRawUpdateSender{}

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
	)

	//act
	err = checker.Check(ctx)

	//assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem batches: got %d, want %d", len(messageSender.problems), 1)
	}

	if len(messageSender.problems) == 1 {
		gotProblems := messageSender.problems[0]

		if len(gotProblems) != 1 {
			t.Errorf("unexpected problems count: got %d, want %d", len(gotProblems), 1)
		}

		if len(gotProblems) == 1 {
			assertProblem(t, gotProblems[0], trackedURL, "stackoverflow returned unexpected status: 502 Bad Gateway", []int64{1})
		}
	}
}

func TestChecker_Check_StackOverflowInvalidBodyDoesNotCrash(t *testing.T) {
	//arrange
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := schedulerlink.NewService()
	subscriptionService := newTestSubscriptionService()
	messageSender := &fakeRawUpdateSender{}

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

	expectedProblem := "decode stackoverflow response: json: cannot unmarshal string into Go struct field QuestionResponse.items of type []stackoverflow.Question"

	stackClient := fakeLinkClient{
		linkType: schedulerlink.TypeStackOverflow,
		getNewEvents: func(ctx context.Context, link schedulerlink.SchedulerLink, since time.Time) ([]update.Event, error) {
			if !since.Equal(lastUpdated) {
				return nil, errors.New("unexpected since")
			}

			return nil, errors.New(expectedProblem)
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
	)

	//act
	err = checker.Check(ctx)

	//assert
	if err != nil {
		t.Errorf("unexpected checker error: %v", err)
	}

	if len(messageSender.updates) != 0 {
		t.Errorf("unexpected sent updates: got %d, want %d", len(messageSender.updates), 0)
	}

	if len(messageSender.problems) != 1 {
		t.Errorf("unexpected sent problem batches: got %d, want %d", len(messageSender.problems), 1)
	}

	if len(messageSender.problems) == 1 {
		gotProblems := messageSender.problems[0]

		if len(gotProblems) != 1 {
			t.Errorf("unexpected problems count: got %d, want %d", len(gotProblems), 1)
		}

		if len(gotProblems) == 1 {
			assertProblem(t, gotProblems[0], trackedURL, expectedProblem, []int64{1})
		}
	}
}

func assertProblem(
	t *testing.T,
	got sender.Problem,
	wantURL string,
	wantMessagePart string,
	wantChatIDs []int64,
) {
	t.Helper()

	if got.URL != wantURL {
		t.Errorf("unexpected problem url: got %q, want %q", got.URL, wantURL)
	}

	if !strings.Contains(got.Message, wantMessagePart) {
		t.Errorf("unexpected problem message: got %q, want to contain %q", got.Message, wantMessagePart)
	}

	slices.Sort(got.ChatIDs)
	slices.Sort(wantChatIDs)

	if !slices.Equal(got.ChatIDs, wantChatIDs) {
		t.Errorf("unexpected problem chat ids: got %#v, want %#v", got.ChatIDs, wantChatIDs)
	}
}

func newTestSubscriptionService() service.SubscriptionService {
	chatRepository := memory.NewChatRepository()
	subscriptionRepository := memory.NewSubscriptionRepository()
	return service.NewSubscriptionService(false, chatRepository, subscriptionRepository, nil, nil)
}
