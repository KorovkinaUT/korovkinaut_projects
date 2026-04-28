package receiver

import (
	"context"
	"log/slog"

	botkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/bot"
)

type BotKafkaReceiver struct {
	consumer *botkafka.Consumer
}

func NewKafkaReceiver(
	brokers []string,
	topic string,
	groupID string,
	dlqTopic string,
	maxAttempts int,
	sendMessage func(chatID int64, text string) error,
) *BotKafkaReceiver {
	return &BotKafkaReceiver{
		consumer: botkafka.NewConsumer(
			brokers,
			topic,
			groupID,
			dlqTopic,
			maxAttempts,
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
