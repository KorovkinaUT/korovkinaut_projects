package sender

import (
	"context"
	"encoding/json"
	"fmt"

	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	agentkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/agent"

	"github.com/segmentio/kafka-go"
)

// Sends raw updates to agent over Kafka.
type AgentKafkaSender struct {
	writer *kafka.Writer
	topic  string
}

var _ appsender.RawUpdateSender = (*AgentKafkaSender)(nil)

func NewAgentKafkaSender(brokers []string, topic string) *AgentKafkaSender {
	return &AgentKafkaSender{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
		},
		topic: topic,
	}
}

func (s *AgentKafkaSender) SendUpdateEvents(ctx context.Context, msg appsender.RawUpdateEvents) error {
	kafkaMsg := agentkafka.Message{
		Kind:         agentkafka.MessageKindUpdateEvents,
		UpdateEvents: msg,
	}

	if err := s.send(ctx, []byte(msg.URL), kafkaMsg); err != nil {
		return fmt.Errorf("send raw update events over kafka: %w", err)
	}

	return nil
}

func (s *AgentKafkaSender) SendProblems(ctx context.Context, msg []appsender.Problem) error {
	kafkaMsg := agentkafka.Message{
		Kind:     agentkafka.MessageKindProblems,
		Problems: msg,
	}

	if err := s.send(ctx, []byte("problems"), kafkaMsg); err != nil {
		return fmt.Errorf("send raw problems over kafka: %w", err)
	}

	return nil
}

func (s *AgentKafkaSender) Close() error {
	if err := s.writer.Close(); err != nil {
		return fmt.Errorf("close kafka writer: %w", err)
	}

	return nil
}

func (s *AgentKafkaSender) send(ctx context.Context, key []byte, msg agentkafka.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal raw kafka message: %w", err)
	}

	err = s.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("write message to kafka topic %q: %w", s.topic, err)
	}

	return nil
}
