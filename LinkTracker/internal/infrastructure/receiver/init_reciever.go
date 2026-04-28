package receiver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

type MessageReceiver interface {
	Start(ctx context.Context, logger *slog.Logger, stop context.CancelFunc)
	Shutdown(ctx context.Context) error
}

func NewMessageReceiver(
	transport string,
	botCfg *config.BotConfig,
	kafkaCfg *config.KafkaConfig,
	sendMessage func(chatID int64, text string) error,
) (MessageReceiver, error) {
	switch strings.ToUpper(transport) {
	case "HTTP":
		return NewHTTPReceiver(
			botCfg.BotAddress(),
			sendMessage,
		), nil

	case "KAFKA":
		return NewKafkaReceiver(
			kafkaCfg.Brokers,
			kafkaCfg.UpdatesTopic,
			kafkaCfg.UpdatesConsumerGroup,
			kafkaCfg.DLQTopic,
			kafkaCfg.ConsumerMaxAttempts,
			sendMessage,
		), nil

	default:
		return nil, fmt.Errorf("unknown updates transport: %s", transport)
	}
}
