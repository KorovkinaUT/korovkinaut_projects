package receiver

import (
	"context"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
)

type BotHTTPReceiver struct {
	server *bothttp.Server
}

var _ MessageReceiver = (*BotHTTPReceiver)(nil)

func NewBotHTTPReceiver(
	address string,
	rateLimitCfg *config.RateLimitConfig,
	sendMessage func(chatID int64, text string) error,
) *BotHTTPReceiver {
	return &BotHTTPReceiver{
		server: bothttp.NewServer(address, sendMessage, rateLimitCfg),
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