package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
)

type Untrack struct {
	parser     *schedulerlink.Service
	removeLink func(ctx context.Context, chatID int64, request scrapperhttp.RemoveLinkRequest) (scrapperhttp.LinkResponse, error)
}

func NewUntrack(
	parser *schedulerlink.Service,
	removeLink func(ctx context.Context, chatID int64, request scrapperhttp.RemoveLinkRequest) (scrapperhttp.LinkResponse, error),
) Untrack {
	return Untrack{
		parser:     parser,
		removeLink: removeLink,
	}
}

func (h Untrack) Command() string { return "untrack" }
func (h Untrack) Description() string {
	return "прекратить отслеживание ссылки"
}

func (h Untrack) Handle(ctx context.Context, msg *tgbotapi.Message) string {
	if msg == nil || msg.Chat == nil {
		return "Не удалось определить чат."
	}

	chatID := msg.Chat.ID
	args := strings.Fields(strings.TrimSpace(msg.Text))

	if len(args) < 2 {
		return "Не достаточно аргументов. Использование: /untrack <url>."
	}

	link := args[1]
	if _, err := h.parser.Parse(strings.TrimSpace(link)); err != nil {
		return "Некорректная ссылка."
	}

	_, err := h.removeLink(ctx, chatID, scrapperhttp.RemoveLinkRequest{Link: link})
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			return "Не удалось найти чат."
		}
		return fmt.Sprintf("Не удалось прекратить отслеживание ссылки:\n%s", link)
	}

	return fmt.Sprintf("Ссылка удалена из отслеживания:\n%s", link)
}
