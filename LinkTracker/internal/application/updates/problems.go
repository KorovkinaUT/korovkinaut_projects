package updates

import (
	"fmt"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
)

// Struct for erros occured during updates processing
type problem struct {
	URL     string
	Message string
	ChatIDs []int64
}

func buildProblemsMessages(problems []problem, nextID func() int64) []sender.ProblemsMessage {
	if len(problems) == 0 {
		return nil
	}

	problemsByChat := make(map[int64][]problem)
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

func formatProblemsMessage(problems []problem) string {
	lines := make([]string, 0, len(problems)*2)

	for _, problem := range problems {
		lines = append(lines, fmt.Sprintf("• Ссылка: %s\n  Причина: %s", problem.URL, problem.Message))
		lines = append(lines, "")
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
