package dispatcher

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
)

type List struct {
	listLinks func(ctx context.Context, chatID int64) (scrapperhttp.ListLinksResponse, error)
}

func NewList(listLinks func(ctx context.Context, chatID int64) (scrapperhttp.ListLinksResponse, error)) List {
	return List{
		listLinks: listLinks,
	}
}

func (h List) Command() string { return "list" }
func (h List) Description() string {
	return "вывести список всех отслеживаемых ссылок"
}

func (h List) Handle(ctx context.Context, msg *tgbotapi.Message) string {
	if msg == nil || msg.Chat == nil {
		return "Не удалось определить чат."
	}

	resp, err := h.listLinks(ctx, msg.Chat.ID)
	if err != nil {
		if errors.Is(err, repository.ErrChatNotFound) {
			return "Список отслеживаемых ссылок пуст."
		}
		return "Не удалось получить список ссылок."
	}

	tagFilter := parseListTag(msg.CommandArguments())

	var b strings.Builder
	b.WriteString("Отслеживаемые ссылки:\n")

	count := 0
	for _, link := range resp.Links {
		if tagFilter != "" && !hasTag(link.Tags, tagFilter) {
			continue
		}

		count++
		b.WriteString(strconv.Itoa(count))
		b.WriteString(". ")
		b.WriteString(link.URL)

		if len(link.Tags) > 0 {
			b.WriteString(" [теги: ")
			b.WriteString(strings.Join(link.Tags, ", "))
			b.WriteString("]")
		}

		b.WriteString("\n")
	}

	if count == 0 {
		return "Список отслеживаемых ссылок пуст."
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func parseListTag(args string) string {
	tag := strings.TrimSpace(args)
	if tag == "" {
		return ""
	}

	fields := strings.Fields(tag)
	if len(fields) == 0 {
		return ""
	}

	return fields[0]
}

func hasTag(tags []string, target string) bool {
	return slices.Contains(tags, target)
}
