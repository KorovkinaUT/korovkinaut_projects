package updates

import (
	"fmt"
	"strings"

	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

type StackOverflowFormatter struct{}

func (f StackOverflowFormatter) Type() schedulerlink.LinkType {
	return schedulerlink.TypeStackOverflow
}

func (f StackOverflowFormatter) Format(rawURL string, events []update.Event) (string, error) {
	if len(events) == 0 {
		return "", fmt.Errorf("no stackoverflow events to format")
	}

	lines := make([]string, 0, len(events)*2)

	for _, event := range events {
		stackOverflowEvent, ok := event.(update.StackOverflowEvent)
		if !ok {
			return "", fmt.Errorf("expected stackoverflow event, got %T", event)
		}

		lines = append(lines, formatStackOverflowEvent(stackOverflowEvent))
		lines = append(lines, "")
	}

	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

func formatStackOverflowEvent(event update.StackOverflowEvent) string {
	return fmt.Sprintf(
		"• %s\n  Вопрос: %s\n  Пользователь: %s\n  Время создания: %s\n  Превью: %s",
		stackOverflowEventLabel(event.Type),
		strings.TrimSpace(event.QuestionTitle),
		strings.TrimSpace(event.Username),
		formatEventTime(event.CreatedAt()),
		strings.TrimSpace(event.Preview),
	)
}

func stackOverflowEventLabel(eventType update.StackOverflowEventType) string {
	switch eventType {
	case update.StackOverflowEventAnswer:
		return "Новый ответ"
	case update.StackOverflowEventComment:
		return "Новый комментарий"
	default:
		return "Новое StackOverflow обновление"
	}
}
