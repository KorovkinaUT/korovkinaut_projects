package main

import (
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dispatcher"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/telegram"
)

func registerBotCommands(tg *telegram.Client, handlers []dispatcher.Handler, logger *slog.Logger) {
	cmdMap := make(map[string]string, len(handlers))

	for _, h := range handlers {
		cmdMap[h.Command()] = h.Description()
	}

	if err := tg.SetMyCommands(cmdMap); err != nil {
		logger.Warn("failed to set bot commands", "error", err)
	}
}
