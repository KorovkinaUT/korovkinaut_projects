package updates

import (
	"fmt"
	"strings"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

type GitHubFormatter struct{}

func (f GitHubFormatter) Type() schedulerlink.LinkType {
	return schedulerlink.TypeGitHub
}

func (f GitHubFormatter) Format(rawURL string, events []update.Event) (string, error) {
	if len(events) == 0 {
		return "", fmt.Errorf("no github events to format")
	}

	lines := make([]string, 0, len(events)*2)

	for _, event := range events {
		githubEvent, ok := event.(update.GitHubEvent)
		if !ok {
			return "", fmt.Errorf("expected github event, got %T", event)
		}

		lines = append(lines, formatGitHubEvent(githubEvent))
		lines = append(lines, "")
	}

	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

func formatGitHubEvent(event update.GitHubEvent) string {
	return fmt.Sprintf(
		"• %s\n  Название: %s\n  Пользователь: %s\n  Время создания: %s\n  Превью: %s",
		gitHubEventLabel(event.Type),
		strings.TrimSpace(event.Title),
		strings.TrimSpace(event.Username),
		formatEventTime(event.CreatedAt()),
		strings.TrimSpace(event.Preview),
	)
}

func gitHubEventLabel(eventType update.GitHubEventType) string {
	switch eventType {
	case update.GitHubEventIssue:
		return "Новый Issue"
	case update.GitHubEventPullRequest:
		return "Новый Pull Request"
	default:
		return "Новое GitHub обновление"
	}
}
