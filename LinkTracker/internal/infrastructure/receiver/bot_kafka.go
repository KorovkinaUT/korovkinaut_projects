package receiver

import (
	"context"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	botkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/bot"
)

type BotKafkaReceiver struct {
	consumer *botkafka.Consumer
}

var _ MessageReceiver = (*BotKafkaReceiver)(nil)

func NewBotKafkaReceiver(
	cfg config.KafkaConfig,
	sendMessage func(chatID int64, text string) error,
) *BotKafkaReceiver {
	return &BotKafkaReceiver{
		consumer: botkafka.NewConsumer(
			cfg,
			sendMessage,
		),
	}
}

func (r *BotKafkaReceiver) Start(ctx context.Context, logger *slog.Logger, stop context.CancelFunc) {
	go func() {
		if err := r.consumer.Start(ctx, logger); err != nil {
			logger.Error("kafka consumer stopped with error", "error", err)
			stop()
		}
	}()
}

func (r *BotKafkaReceiver) Shutdown(_ context.Context) error {
	return r.consumer.Close()
}
