package receiver

import (
	"context"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	agentkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/agent"
)

type AgentKafkaReciever struct {
	consumer *agentkafka.Consumer
}

func NewAgentKafkaReceiver(
	cfg config.KafkaConfig,
	processor agentkafka.ProcessingService,
	messageSender sender.MessageSender,
) *AgentKafkaReciever {
	return &AgentKafkaReciever{
		consumer: agentkafka.NewConsumer(
			cfg,
			processor,
			messageSender,
		),
	}
}

func (r *AgentKafkaReciever) Start(ctx context.Context, logger *slog.Logger, stop context.CancelFunc) {
	go func() {
		if err := r.consumer.Start(ctx, logger); err != nil {
			logger.Error("agent kafka consumer stopped with error", "error", err)
			stop()
		}
	}()
}

func (r *AgentKafkaReciever) Shutdown() error {
	return r.consumer.Close()
}