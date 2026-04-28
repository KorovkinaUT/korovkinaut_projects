package sender

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
)

const maxMessageLength = 4000

// Sends messages to bot over HTTP.
type BotHTTPSender struct {
	client *bothttp.Client
}

var _ appsender.MessageSender = (*BotHTTPSender)(nil)

func NewHTTPSender(client *bothttp.Client) *BotHTTPSender {
	return &BotHTTPSender{
		client: client,
	}
}

func (s *BotHTTPSender) SendUpdate(ctx context.Context, msg appsender.UpdateMessage) error {
	parts := splitMessageByBullet(msg.Description, maxMessageLength)

	for _, part := range parts {
		err := s.client.SendUpdate(ctx, bothttp.LinkUpdate{
			ID:  msg.ID,
			URL: msg.URL,
			Description: fmt.Sprintf(
				"Появилось обновление по ссылке: %s\n\n%s",
				msg.URL,
				part,
			),
			TgChatIDs: msg.TgChatIDs,
		})
		if err != nil {
			return fmt.Errorf("send update over http: %w", err)
		}
	}

	return nil
}

func (s *BotHTTPSender) SendProblems(ctx context.Context, msg appsender.ProblemsMessage) error {
	parts := splitMessageByBullet(msg.Description, maxMessageLength)

	for _, part := range parts {
		err := s.client.SendUpdate(ctx, bothttp.LinkUpdate{
			ID:  msg.ID,
			URL: "problems",
			Description: fmt.Sprintf(
				"Не удалось проверить некоторые ссылки:\n\n%s",
				part,
			),
			TgChatIDs: msg.TgChatIDs,
		})
		if err != nil {
			return fmt.Errorf("send problems over http: %w", err)
		}
	}

	return nil
}

// Splits update message, if it is bigger than limit for one Telegram message
func splitMessageByBullet(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}

	if runeLen(text) <= limit {
		return []string{text}
	}

	blocks := splitBulletBlocks(text)
	if len(blocks) == 0 {
		return []string{text}
	}

	parts := make([]string, 0)
	current := ""

	for _, block := range blocks {
		if current == "" {
			current = block
			continue
		}

		candidate := current + "\n\n" + block
		if runeLen(candidate) <= limit {
			current = candidate
			continue
		}

		parts = append(parts, current)
		current = block
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// Every new event description starts with •
func splitBulletBlocks(text string) []string {
	rawParts := strings.Split(text, "•")
	blocks := make([]string, 0, len(rawParts))

	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		blocks = append(blocks, "• "+part)
	}

	return blocks
}

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}
