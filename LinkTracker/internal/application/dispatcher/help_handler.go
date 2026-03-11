package dispatcher

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type help struct{}

func (help) Command() string                   { return "help" }
func (help) Description() string               { return "список доступных команд" }
func (help) Handle(_ *tgbotapi.Message) string { return "" } // Dispatcher generates the output of /help
