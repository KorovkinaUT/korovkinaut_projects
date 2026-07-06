package receiver

import (
	"context"
	"strings"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func TestNewMessageReceiver_HTTPTransport_ReturnsCompositeReceiver(t *testing.T) {
	//arrange
	botCfg, kafkaCfg := testReceiverConfigs()
	sendMessage := func(chatID int64, text string) error {
		return nil
	}

	//act
	messageReceiver, err := NewMessageReceiver("HTTP", botCfg, kafkaCfg, sendMessage)

	//assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := messageReceiver.(*BotCompositeReceiver); !ok {
		t.Errorf("unexpected receiver type: got %T, want *CompositeReceiver", messageReceiver)
	}
}

func TestNewMessageReceiver_KafkaTransport_ReturnsKafkaReceiver(t *testing.T) {
	//arrange
	botCfg, kafkaCfg := testReceiverConfigs()
	sendMessage := func(chatID int64, text string) error {
		return nil
	}

	//act
	messageReceiver, err := NewMessageReceiver("KAFKA", botCfg, kafkaCfg, sendMessage)

	//assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	kafkaReceiver, ok := messageReceiver.(*BotKafkaReceiver)
	if !ok {
		t.Errorf("unexpected receiver type: got %T, want *BotKafkaReceiver", messageReceiver)
		return
	}

	if err := kafkaReceiver.Shutdown(context.Background()); err != nil {
		t.Errorf("shutdown kafka receiver: %v", err)
	}
}

func TestNewMessageReceiver_UnknownTransport_ReturnsError(t *testing.T) {
	//arrange
	botCfg, kafkaCfg := testReceiverConfigs()
	sendMessage := func(chatID int64, text string) error {
		return nil
	}

	//act
	messageReceiver, err := NewMessageReceiver("GRPC", botCfg, kafkaCfg, sendMessage)

	//assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "unknown updates transport: GRPC") {
		t.Errorf("unexpected error: got %q", err.Error())
	}

	if messageReceiver != nil {
		t.Errorf("expected nil receiver, got %T", messageReceiver)
	}
}

func testReceiverConfigs() (*config.BotConfig, *config.KafkaConfig) {
	return &config.BotConfig{
			BotHost: "localhost",
			BotPort: 8080,
			RateLimit: &config.RateLimitConfig{
				RPS:   10,
				Burst: 20,
			},
		},
		&config.KafkaConfig{
			Brokers:              []string{"localhost:9092"},
			ProcessedUpdatesTopic:         "link-updates",
			ProcessedUpdatesConsumerGroup: "bot",
			DLQTopic:             "link-tracker-dlq",
			ConsumerMaxAttempts:  3,
		}
}
