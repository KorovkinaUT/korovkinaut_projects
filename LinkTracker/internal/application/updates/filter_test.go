package updates

import (
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func TestFilter_ApplyEvents_StopWord_FiltersEvent(t *testing.T) {
	//arrange
	filter := NewFilter(&config.AgentConfig{
		StopWords:       []string{"spam"},
		MinLength:       5,
		ExcludedAuthors: []string{"bot-user"},
	})

	events := []sender.Event{
		{
			Source:       "github",
			Type:         "issue",
			Title:        "Issue title",
			Author:       "alice",
			Preview:      "This update contains spam content",
			CreationTime: time.Now(),
		},
	}

	//act
	filtered := filter.ApplyEvents(events)

	//assert
	if len(filtered) != 0 {
		t.Errorf("expected no events after filtering, got %d", len(filtered))
	}
}

func TestFilter_ApplyEvents_ExcludedAuthor_FiltersEvent(t *testing.T) {
	//arrange
	filter := NewFilter(&config.AgentConfig{
		StopWords:       []string{"spam"},
		MinLength:       5,
		ExcludedAuthors: []string{"bot-user"},
	})

	events := []sender.Event{
		{
			Source:       "github",
			Type:         "issue",
			Title:        "Issue title",
			Author:       "bot-user",
			Preview:      "Valid update preview",
			CreationTime: time.Now(),
		},
	}

	//act
	filtered := filter.ApplyEvents(events)

	//assert
	if len(filtered) != 0 {
		t.Errorf("expected no events after filtering, got %d", len(filtered))
	}
}

func TestFilter_ApplyEvents_TooShortPreview_FiltersEvent(t *testing.T) {
	//arrange
	filter := NewFilter(&config.AgentConfig{
		StopWords:       []string{"spam"},
		MinLength:       20,
		ExcludedAuthors: []string{"bot-user"},
	})

	events := []sender.Event{
		{
			Source:       "github",
			Type:         "issue",
			Title:        "Issue title",
			Author:       "alice",
			Preview:      "short",
			CreationTime: time.Now(),
		},
	}

	//act
	filtered := filter.ApplyEvents(events)

	//assert
	if len(filtered) != 0 {
		t.Errorf("expected no events after filtering, got %d", len(filtered))
	}
}

func TestFilter_ApplyEvents_ValidEvent_ReturnsEvent(t *testing.T) {
	//arrange
	filter := NewFilter(&config.AgentConfig{
		StopWords:       []string{"spam"},
		MinLength:       5,
		ExcludedAuthors: []string{"bot-user"},
	})

	event := sender.Event{
		Source:       "github",
		Type:         "issue",
		Title:        "Issue title",
		Author:       "alice",
		Preview:      "Valid update preview",
		CreationTime: time.Now(),
	}

	events := []sender.Event{event}

	//act
	filtered := filter.ApplyEvents(events)

	//assert
	if len(filtered) != 1 {
		t.Fatalf("expected one event after filtering, got %d", len(filtered))
	}

	if filtered[0] != event {
		t.Errorf("expected event %+v, got %+v", event, filtered[0])
	}
}

func TestFilter_ApplyProblems_StopWord_FiltersProblem(t *testing.T) {
	//arrange
	filter := NewFilter(&config.AgentConfig{
		StopWords: []string{"spam"},
		MinLength: 5,
	})

	problems := []sender.Problem{
		{
			URL:     "https://github.com/user/repo",
			Message: "request failed because of spam content",
			ChatIDs: []int64{1},
		},
	}

	//act
	filtered := filter.ApplyProblems(problems)

	//assert
	if len(filtered) != 0 {
		t.Errorf("expected no problems after filtering, got %d", len(filtered))
	}
}

func TestFilter_ApplyProblems_TooShortMessage_FiltersProblem(t *testing.T) {
	//arrange
	filter := NewFilter(&config.AgentConfig{
		StopWords: []string{"spam"},
		MinLength: 20,
	})

	problems := []sender.Problem{
		{
			URL:     "https://github.com/user/repo",
			Message: "short",
			ChatIDs: []int64{1},
		},
	}

	//act
	filtered := filter.ApplyProblems(problems)

	//assert
	if len(filtered) != 0 {
		t.Errorf("expected no problems after filtering, got %d", len(filtered))
	}
}

func TestFilter_ApplyProblems_ValidProblem_ReturnsProblem(t *testing.T) {
	//arrange
	filter := NewFilter(&config.AgentConfig{
		StopWords: []string{"spam"},
		MinLength: 5,
	})

	problem := sender.Problem{
		URL:     "https://github.com/user/repo",
		Message: "github returned unexpected status",
		ChatIDs: []int64{1},
	}

	problems := []sender.Problem{problem}

	//act
	filtered := filter.ApplyProblems(problems)

	//assert
	if len(filtered) != 1 {
		t.Fatalf("expected one problem after filtering, got %d", len(filtered))
	}

	if filtered[0].URL != problem.URL {
		t.Errorf("expected problem url %q, got %q", problem.URL, filtered[0].URL)
	}

	if filtered[0].Message != problem.Message {
		t.Errorf("expected problem message %q, got %q", problem.Message, filtered[0].Message)
	}

	if len(filtered[0].ChatIDs) != len(problem.ChatIDs) || filtered[0].ChatIDs[0] != problem.ChatIDs[0] {
		t.Errorf("expected problem chat ids %+v, got %+v", problem.ChatIDs, filtered[0].ChatIDs)
	}
}