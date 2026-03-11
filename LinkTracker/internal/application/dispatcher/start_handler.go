package dispatcher

import (
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
)

type Start struct {
	registerChat func(chatID int64) error
}

func NewStart(registerChat func(chatID int64) error) Start {
	return Start{
		registerChat: registerChat,
	}
}

func (Start) Command() string     { return "start" }
func (Start) Description() string { return "начало работы" }

func (h Start) Handle(msg *tgbotapi.Message) string {
	if msg == nil || msg.Chat == nil {
		return "Не удалось определить чат."
	}

	err := h.registerChat(msg.Chat.ID)
	// If chat is already registered, not return error
	if err != nil && !errors.Is(err, repository.ErrChatAlreadyExists) {
		return "Не удалось зарегистрировать чат."
	}

	return "Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды."
}
