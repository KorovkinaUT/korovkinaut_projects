package updates

import (
	"strings"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

func TestGitHubFormatter_Format_Issue(t *testing.T) {
	// arrange
	formatter := GitHubFormatter{}
	createdAt := time.Date(2026, 4, 4, 12, 30, 0, 0, time.UTC)

	events := []update.Event{
		update.GitHubEvent{
			Type:         update.GitHubEventIssue,
			Title:        "Fix race in checker",
			Username:     "octocat",
			CreationTime: createdAt,
			Preview:      "Need to protect shared state in worker pool",
		},
	}

	// act
	got, err := formatter.Format("https://github.com/user/repo", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	expectedTime := createdAt.Format("02 Jan 2006 15:04")

	if !strings.Contains(got, "• ") {
		t.Errorf("formatted message must contain bullet prefix, got %q", got)
	}

	if !strings.Contains(got, "Issue") {
		t.Errorf("formatted message must contain issue label, got %q", got)
	}

	if !strings.Contains(got, "Fix race in checker") {
		t.Errorf("formatted message must contain title, got %q", got)
	}

	if !strings.Contains(got, "octocat") {
		t.Errorf("formatted message must contain username, got %q", got)
	}

	if !strings.Contains(got, expectedTime) {
		t.Errorf("formatted message must contain formatted creation time, got %q", got)
	}

	if !strings.Contains(got, "Need to protect shared state in worker pool") {
		t.Errorf("formatted message must contain preview, got %q", got)
	}
}

func TestGitHubFormatter_Format_PullRequest(t *testing.T) {
	// arrange
	formatter := GitHubFormatter{}
	createdAt := time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)

	events := []update.Event{
		update.GitHubEvent{
			Type:         update.GitHubEventPullRequest,
			Title:        "Add batch processing",
			Username:     "alice",
			CreationTime: createdAt,
			Preview:      "Implements configurable workers count",
		},
	}

	// act
	got, err := formatter.Format("https://github.com/user/repo", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	expectedTime := createdAt.Format("02 Jan 2006 15:04")

	if !strings.Contains(got, "• ") {
		t.Errorf("formatted message must contain bullet prefix, got %q", got)
	}

	if !strings.Contains(got, "Pull Request") {
		t.Errorf("formatted message must contain pull request label, got %q", got)
	}

	if !strings.Contains(got, "Add batch processing") {
		t.Errorf("formatted message must contain title, got %q", got)
	}

	if !strings.Contains(got, "alice") {
		t.Errorf("formatted message must contain username, got %q", got)
	}

	if !strings.Contains(got, expectedTime) {
		t.Errorf("formatted message must contain formatted creation time, got %q", got)
	}

	if !strings.Contains(got, "Implements configurable workers count") {
		t.Errorf("formatted message must contain preview, got %q", got)
	}
}

func TestGitHubFormatter_Format_UnknownEventType(t *testing.T) {
	// arrange
	formatter := GitHubFormatter{}
	createdAt := time.Date(2026, 2, 10, 8, 15, 0, 0, time.UTC)

	events := []update.Event{
		update.GitHubEvent{
			Type:         update.GitHubEventType("unknown"),
			Title:        "Some update",
			Username:     "bob",
			CreationTime: createdAt,
			Preview:      "Preview text",
		},
	}

	// act
	got, err := formatter.Format("https://github.com/user/repo", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	expectedTime := createdAt.Format("02 Jan 2006 15:04")

	if !strings.Contains(got, "Some update") {
		t.Errorf("formatted message must contain title, got %q", got)
	}

	if !strings.Contains(got, "bob") {
		t.Errorf("formatted message must contain username, got %q", got)
	}

	if !strings.Contains(got, expectedTime) {
		t.Errorf("formatted message must contain formatted creation time, got %q", got)
	}

	if !strings.Contains(got, "Preview text") {
		t.Errorf("formatted message must contain preview, got %q", got)
	}
}

func TestGitHubFormatter_Format_MultipleEvents(t *testing.T) {
	// arrange
	formatter := GitHubFormatter{}

	events := []update.Event{
		update.GitHubEvent{
			Type:         update.GitHubEventIssue,
			Title:        "First issue",
			Username:     "alice",
			CreationTime: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			Preview:      "First preview",
		},
		update.GitHubEvent{
			Type:         update.GitHubEventPullRequest,
			Title:        "Second pr",
			Username:     "bob",
			CreationTime: time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
			Preview:      "Second preview",
		},
	}

	// act
	got, err := formatter.Format("https://github.com/user/repo", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	if !strings.Contains(got, "Issue") {
		t.Errorf("formatted output must contain first event label, got %q", got)
	}

	if !strings.Contains(got, "First issue") {
		t.Errorf("formatted output must contain first event title, got %q", got)
	}

	if !strings.Contains(got, "Pull Request") {
		t.Errorf("formatted output must contain second event label, got %q", got)
	}

	if !strings.Contains(got, "Second pr") {
		t.Errorf("formatted output must contain second event title, got %q", got)
	}
}

func TestGitHubFormatter_Format_WrongEventType(t *testing.T) {
	// arrange
	formatter := GitHubFormatter{}
	events := []update.Event{
		update.StackOverflowEvent{
			Type:          update.StackOverflowEventAnswer,
			QuestionTitle: "Question",
			Username:      "user",
			CreationTime:  time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
			Preview:       "Preview",
		},
	}

	// act
	got, err := formatter.Format("https://github.com/user/repo", events)

	// assert
	if err == nil {
		t.Errorf("expected error for wrong event type, got nil")
	}

	if got != "" {
		t.Errorf("expected empty formatted string, got %q", got)
	}
}
