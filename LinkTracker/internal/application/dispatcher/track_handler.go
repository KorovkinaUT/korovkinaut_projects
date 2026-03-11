package dispatcher

import (
	"errors"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
)

type Track struct {
	states  *StateStorage // dialogs states
	parser  *schedulerlink.Service // checks link correctness
	addLink func(chatID int64, request scrapperhttp.AddLinkRequest) (scrapperhttp.LinkResponse, error)
}

func NewTrack(
	parser *schedulerlink.Service,
	addLink func(chatID int64, request scrapperhttp.AddLinkRequest) (scrapperhttp.LinkResponse, error),
) Track {
	return Track{
		states:  NewStateStorage(),
		parser:  parser,
		addLink: addLink,
	}
}

func (h Track) Command() string     { return "track" }
func (h Track) Description() string { return "начать отслеживание ссылки" }

func (h Track) States() *StateStorage {
	return h.states
}

// Hanlde /track command
func (h Track) Handle(msg *tgbotapi.Message) string {
	if msg == nil || msg.Chat == nil {
		return "Не удалось определить чат."
	}

	h.states.Set(msg.Chat.ID, TrackDialog{
		State: StateWaitingTrackLink,
	})

	return "Отправьте ссылку для отслеживания."
}

// Handle other massages after /track
func (h Track) HandleDialog(msg *tgbotapi.Message, dialog TrackDialog) string {
	chatID := msg.Chat.ID

	switch dialog.State {
	case StateWaitingTrackLink:
		if msg.IsCommand() {
			h.states.Reset(chatID)
			return "Процесс отслеживания отменён."
		}

		if _, err := h.parser.Parse(strings.TrimSpace(msg.Text)); err != nil {
			return "Некорректная ссылка."
		}

		h.states.Set(chatID, TrackDialog{
			State: StateWaitingTrackTags,
			Link:  strings.TrimSpace(msg.Text),
		})

		return "Введите теги через запятую, введите /cancel, чтобы не добавлять теги."

	case StateWaitingTrackTags:
		// /cancel
		if msg.IsCommand() {
			return h.finishDialog(chatID, dialog.Link, []string{})
		}

		tags := parseTrackTags(msg.Text)
		return h.finishDialog(chatID, dialog.Link, tags)

	default:
		h.states.Reset(chatID)
		return "Не удалось обработать состояние диалога."
	}
}

func (h Track) finishDialog(chatID int64, link string, tags []string) string {
	h.states.Reset(chatID)

	_, err := h.addLink(chatID, scrapperhttp.AddLinkRequest{
		Link: link,
		Tags: tags,
	})
	if err != nil {
		if errors.Is(err, repository.ErrLinkAlreadyTracked) {
			return "Ссылка уже отслеживается"
		}
		if errors.Is(err, repository.ErrChatNotFound) {
			return "Не удалось найти чат."
		}
		return "Не удалось сохранить ссылку."
	}

	return "Ссылка добавлена в отслеживание."
}

func parseTrackTags(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}

	parts := strings.Split(trimmed, ",")
	tags := make([]string, 0, len(parts))

	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		tags = append(tags, tag)
	}

	return tags
}
