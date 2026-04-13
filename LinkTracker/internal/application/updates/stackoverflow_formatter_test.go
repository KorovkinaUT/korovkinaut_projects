package updates

import (
	"strings"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

func TestStackOverflowFormatter_Format_Answer(t *testing.T) {
	// arrange
	formatter := StackOverflowFormatter{}
	createdAt := time.Date(2026, 4, 4, 12, 30, 0, 0, time.UTC)

	events := []update.Event{
		update.StackOverflowEvent{
			Type:          update.StackOverflowEventAnswer,
			QuestionTitle: "How to use context in Go?",
			Username:      "alice",
			CreationTime:  createdAt,
			Preview:       "You should pass context through call chain",
		},
	}

	// act
	got, err := formatter.Format("https://stackoverflow.com/questions/123/test", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	expectedTime := createdAt.Format("02 Jan 2006 15:04")

	if !strings.Contains(got, "• ") {
		t.Errorf("formatted message must contain bullet prefix, got %q", got)
	}

	if !strings.Contains(got, "ответ") {
		t.Errorf("formatted message must contain answer label, got %q", got)
	}

	if !strings.Contains(got, "How to use context in Go?") {
		t.Errorf("formatted message must contain question title, got %q", got)
	}

	if !strings.Contains(got, "alice") {
		t.Errorf("formatted message must contain username, got %q", got)
	}

	if !strings.Contains(got, expectedTime) {
		t.Errorf("formatted message must contain formatted creation time, got %q", got)
	}

	if !strings.Contains(got, "You should pass context through call chain") {
		t.Errorf("formatted message must contain preview, got %q", got)
	}
}

func TestStackOverflowFormatter_Format_Comment(t *testing.T) {
	// arrange
	formatter := StackOverflowFormatter{}
	createdAt := time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC)

	events := []update.Event{
		update.StackOverflowEvent{
			Type:          update.StackOverflowEventComment,
			QuestionTitle: "What is interface in Go?",
			Username:      "bob",
			CreationTime:  createdAt,
			Preview:       "A small clarification about empty interface",
		},
	}

	// act
	got, err := formatter.Format("https://stackoverflow.com/questions/123/test", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	expectedTime := createdAt.Format("02 Jan 2006 15:04")

	if !strings.Contains(got, "• ") {
		t.Errorf("formatted message must contain bullet prefix, got %q", got)
	}

	if !strings.Contains(got, "комментарий") {
		t.Errorf("formatted message must contain comment label, got %q", got)
	}

	if !strings.Contains(got, "What is interface in Go?") {
		t.Errorf("formatted message must contain question title, got %q", got)
	}

	if !strings.Contains(got, "bob") {
		t.Errorf("formatted message must contain username, got %q", got)
	}

	if !strings.Contains(got, expectedTime) {
		t.Errorf("formatted message must contain formatted creation time, got %q", got)
	}

	if !strings.Contains(got, "A small clarification about empty interface") {
		t.Errorf("formatted message must contain preview, got %q", got)
	}
}

func TestStackOverflowFormatter_Format_UnknownEventType(t *testing.T) {
	// arrange
	formatter := StackOverflowFormatter{}
	createdAt := time.Date(2026, 2, 10, 8, 15, 0, 0, time.UTC)

	events := []update.Event{
		update.StackOverflowEvent{
			Type:          update.StackOverflowEventType("unknown"),
			QuestionTitle: "Some question",
			Username:      "carol",
			CreationTime:  createdAt,
			Preview:       "Preview text",
		},
	}

	// act
	got, err := formatter.Format("https://stackoverflow.com/questions/123/test", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	expectedTime := createdAt.Format("02 Jan 2006 15:04")

	if !strings.Contains(got, "Some question") {
		t.Errorf("formatted message must contain question title, got %q", got)
	}

	if !strings.Contains(got, "carol") {
		t.Errorf("formatted message must contain username, got %q", got)
	}

	if !strings.Contains(got, expectedTime) {
		t.Errorf("formatted message must contain formatted creation time, got %q", got)
	}

	if !strings.Contains(got, "Preview text") {
		t.Errorf("formatted message must contain preview, got %q", got)
	}
}

func TestStackOverflowFormatter_Format_MultipleEvents(t *testing.T) {
	// arrange
	formatter := StackOverflowFormatter{}

	events := []update.Event{
		update.StackOverflowEvent{
			Type:          update.StackOverflowEventAnswer,
			QuestionTitle: "First question",
			Username:      "alice",
			CreationTime:  time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			Preview:       "First preview",
		},
		update.StackOverflowEvent{
			Type:          update.StackOverflowEventComment,
			QuestionTitle: "Second question",
			Username:      "bob",
			CreationTime:  time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
			Preview:       "Second preview",
		},
	}

	// act
	got, err := formatter.Format("https://stackoverflow.com/questions/123/test", events)

	// assert
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}

	if !strings.Contains(got, "ответ") {
		t.Errorf("formatted output must contain first event label, got %q", got)
	}

	if !strings.Contains(got, "First question") {
		t.Errorf("formatted output must contain first event question title, got %q", got)
	}

	if !strings.Contains(got, "комментарий") {
		t.Errorf("formatted output must contain second event label, got %q", got)
	}

	if !strings.Contains(got, "Second question") {
		t.Errorf("formatted output must contain second event question title, got %q", got)
	}
}

func TestStackOverflowFormatter_Format_WrongEventType(t *testing.T) {
	// arrange
	formatter := StackOverflowFormatter{}
	events := []update.Event{
		update.GitHubEvent{
			Type:         update.GitHubEventIssue,
			Title:        "Issue",
			Username:     "user",
			CreationTime: time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
			Preview:      "Preview",
		},
	}

	// act
	got, err := formatter.Format("https://stackoverflow.com/questions/123/test", events)

	// assert
	if err == nil {
		t.Errorf("expected error for wrong event type, got nil")
	}

	if got != "" {
		t.Errorf("expected empty formatted string, got %q", got)
	}
}