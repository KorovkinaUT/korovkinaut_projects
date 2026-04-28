package receiver

import (
	"context"
	"log/slog"

	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
)

type BotHTTPReceiver struct {
	server *bothttp.Server
}

func NewHTTPReceiver(
	address string,
	sendMessage func(chatID int64, text string) error,
) *BotHTTPReceiver {
	return &BotHTTPReceiver{
		server: bothttp.NewServer(address, sendMessage),
	}
}

func (r *BotHTTPReceiver) Start(_ context.Context, logger *slog.Logger, stop context.CancelFunc) {
	go func() {
		if err := r.server.Start(logger); err != nil {
			logger.Error("bot http server failed", "error", err)
			stop()
		}
	}()
}

func (r *BotHTTPReceiver) Shutdown(ctx context.Context) error {
	return r.server.Shutdown(ctx)
}