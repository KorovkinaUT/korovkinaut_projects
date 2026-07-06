package service

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

// Interface for AI-agent, which summarizes descriptions
type Summarizer interface {
	Summarize(ctx context.Context, text string) (string, error)
}

// Interface for filtering updates
type Filter interface {
	ApplyEvents(events []sender.Event) []sender.Event
	ApplyProblems(problems []sender.Problem) []sender.Problem
}

// Interface for updates/formatter
type Formatter interface {
	Type() schedulerlink.LinkType
	Format(rawURL string, events []update.Event) (string, error)
}

// Service for filtering and making UpdateMessages from RawUpdates
type UpdatesProcessingService struct {
	filter        Filter
	summarizer    Summarizer
	nextMessageID atomic.Int64
	formatters    map[string]Formatter
}

func NewUpdatesProcessingService(
	filter Filter,
	summarizer Summarizer,
	formatters []Formatter,
) *UpdatesProcessingService {
	formattersBySource := make(map[string]Formatter, len(formatters))
	for _, formatter := range formatters {
		formattersBySource[string(formatter.Type())] = formatter
	}

	return &UpdatesProcessingService{
		filter:     filter,
		summarizer: summarizer,
		formatters: formattersBySource,
	}
}

func (s *UpdatesProcessingService) ProcessUpdateEvents(
	ctx context.Context,
	msg *sender.RawUpdateEvents,
) (*sender.UpdateMessage, error) {
	events := s.filter.ApplyEvents(msg.Events)
	if len(events) == 0 {
		return nil, nil
	}

	processedEvents := make([]sender.Event, 0, len(events))
	for _, event := range events {
		processedEvent, err := s.processEvent(ctx, event)
		if err != nil {
			return nil, err
		}

		processedEvents = append(processedEvents, processedEvent)
	}

	if len(events) == 0 {
		return nil, nil
	}
	description, err := s.formatEvents(msg.URL, processedEvents)
	if err != nil {
		return nil, err
	}

	return &sender.UpdateMessage{
		ID:          s.nextMessageID.Add(1),
		URL:         msg.URL,
		Description: description,
		TgChatIDs:   msg.TgChatIDs,
	}, nil
}

func (s *UpdatesProcessingService) ProcessProblems(
	ctx context.Context,
	problems []sender.Problem,
) ([]sender.ProblemsMessage, error) {
	problems = s.filter.ApplyProblems(problems)
	if len(problems) == 0 {
		return nil, nil
	}

	processedProblems := make([]sender.Problem, 0, len(problems))
	for _, problem := range problems {
		processedProblem, err := s.processProblem(ctx, problem)
		if err != nil {
			return nil, err
		}

		processedProblems = append(processedProblems, processedProblem)
	}

	if len(problems) == 0 {
		return nil, nil
	}
	messages := buildProblemsMessages(processedProblems, func() int64 {
		return s.nextMessageID.Add(1)
	})

	return messages, nil
}

func (s *UpdatesProcessingService) processEvent(ctx context.Context, event sender.Event) (sender.Event, error) {
	summary, err := s.summarizer.Summarize(ctx, event.Preview)
	if err != nil {
		return sender.Event{}, err
	}

	event.Preview = summary

	return event, nil
}

func (s *UpdatesProcessingService) processProblem(
	ctx context.Context,
	problem sender.Problem,
) (sender.Problem, error) {
	summary, err := s.summarizer.Summarize(ctx, problem.Message)
	if err != nil {
		return sender.Problem{}, err
	}

	problem.Message = summary

	return problem, nil
}

func (s *UpdatesProcessingService) formatEvents(rawURL string, events []sender.Event) (string, error) {
	source := events[0].Source
	formatter, ok := s.formatters[source]
	if !ok {
		return "", fmt.Errorf("no formatter registered for source %q", source)
	}

	domainEvents, err := makeDomainEvents(events)
	if err != nil {
		return "", fmt.Errorf("make domain events for source %q: %w", source, err)
	}

	description, err := formatter.Format(rawURL, domainEvents)
	if err != nil {
		return "", fmt.Errorf("format events for source %q: %w", source, err)
	}

	return description, nil
}

func formatProblemsMessage(problems []sender.Problem) string {
	lines := make([]string, 0, len(problems)*2)

	for _, problem := range problems {
		lines = append(lines, fmt.Sprintf("• Ссылка: %s\n  Причина: %s", problem.URL, problem.Message))
		lines = append(lines, "")
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func buildProblemsMessages(problems []sender.Problem, nextID func() int64) []sender.ProblemsMessage {
	if len(problems) == 0 {
		return nil
	}

	problemsByChat := make(map[int64][]sender.Problem)
	for _, problem := range problems {
		for _, chatID := range problem.ChatIDs {
			problemsByChat[chatID] = append(problemsByChat[chatID], problem)
		}
	}

	messages := make([]sender.ProblemsMessage, 0, len(problemsByChat))
	for chatID, chatProblems := range problemsByChat {
		messages = append(messages, sender.ProblemsMessage{
			ID:          nextID(),
			Description: formatProblemsMessage(chatProblems),
			TgChatIDs:   []int64{chatID},
		})
	}

	return messages
}

func makeDomainEvents(events []sender.Event) ([]update.Event, error) {
	result := make([]update.Event, 0, len(events))

	for _, event := range events {
		domainEvent, err := makeDomainEvent(event)
		if err != nil {
			return nil, err
		}

		result = append(result, domainEvent)
	}

	return result, nil
}

func makeDomainEvent(event sender.Event) (update.Event, error) {
	switch event.Source {
	case string(schedulerlink.TypeGitHub):
		return makeGitHubEvent(event), nil
	case string(schedulerlink.TypeStackOverflow):
		return makeStackOverflowEvent(event), nil
	default:
		return nil, fmt.Errorf("unsupported event source %q", event.Source)
	}
}

func makeGitHubEvent(event sender.Event) update.GitHubEvent {
	return update.GitHubEvent{
		Type:         update.GitHubEventType(event.Type),
		Title:        strings.TrimSpace(event.Title),
		Username:     strings.TrimSpace(event.Author),
		CreationTime: event.CreationTime,
		Preview:      strings.TrimSpace(event.Preview),
	}
}

func makeStackOverflowEvent(event sender.Event) update.StackOverflowEvent {
	return update.StackOverflowEvent{
		Type:          update.StackOverflowEventType(event.Type),
		QuestionTitle: strings.TrimSpace(event.Title),
		Username:      strings.TrimSpace(event.Author),
		CreationTime:  event.CreationTime,
		Preview:       strings.TrimSpace(event.Preview),
	}
}
