package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	botkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/bot"

	"github.com/segmentio/kafka-go"
)

// Sends messages to bot over Kafka.
type BotKafkaSender struct {
	writer *kafka.Writer
	topic  string
}

var _ appsender.MessageSender = (*BotKafkaSender)(nil)

func NewKafkaSender(brokers []string, topic string) *BotKafkaSender {
	return &BotKafkaSender{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
		},
		topic: topic,
	}
}

func (s *BotKafkaSender) SendUpdate(ctx context.Context, msg appsender.UpdateMessage) error {
	parts := splitMessageByBullet(msg.Description, maxMessageLength)

	for _, part := range parts {
		update := botkafka.LinkUpdate{
			ID:  msg.ID,
			URL: msg.URL,
			Description: fmt.Sprintf(
				"Появилось обновление по ссылке: %s\n\n%s",
				msg.URL,
				part,
			),
			TgChatIDs: msg.TgChatIDs,
		}

		if err := s.sendUpdate(ctx, update); err != nil {
			return fmt.Errorf("send update over kafka: %w", err)
		}
	}

	return nil
}

func (s *BotKafkaSender) SendProblems(ctx context.Context, msg appsender.ProblemsMessage) error {
	parts := splitMessageByBullet(msg.Description, maxMessageLength)

	for _, part := range parts {
		update := botkafka.LinkUpdate{
			ID:  msg.ID,
			URL: "problems",
			Description: fmt.Sprintf(
				"Не удалось проверить некоторые ссылки:\n\n%s",
				part,
			),
			TgChatIDs: msg.TgChatIDs,
		}

		if err := s.sendUpdate(ctx, update); err != nil {
			return fmt.Errorf("send problems over kafka: %w", err)
		}
	}

	return nil
}

func (s *BotKafkaSender) Close() error {
	if err := s.writer.Close(); err != nil {
		return fmt.Errorf("close kafka writer: %w", err)
	}

	return nil
}

func (s *BotKafkaSender) sendUpdate(ctx context.Context, msg botkafka.LinkUpdate) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal kafka update message: %w", err)
	}

	err = s.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatInt(msg.ID, 10)),
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("write message to kafka topic %q: %w", s.topic, err)
	}

	return nil
}
