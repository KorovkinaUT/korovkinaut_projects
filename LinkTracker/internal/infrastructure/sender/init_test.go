package sender

import (
	"net/http"
	"strings"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func TestNewMessageSender_HTTPTransport_ReturnsHTTPSender(t *testing.T) {
	//arrange
	cfg := &config.KafkaConfig{
		Brokers:      []string{"localhost:9092"},
		UpdatesTopic: "link-updates",
	}

	//act
	messageSender, err := NewMessageSender("HTTP", cfg, "http://localhost:8080", http.DefaultClient)

	//assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := messageSender.(*BotHTTPSender); !ok {
		t.Errorf("unexpected sender type: got %T, want *BotHTTPSender", messageSender)
	}
}

func TestNewMessageSender_KafkaTransport_ReturnsKafkaSender(t *testing.T) {
	//arrange
	cfg := &config.KafkaConfig{
		Brokers:      []string{"localhost:9092"},
		UpdatesTopic: "link-updates",
	}

	//act
	messageSender, err := NewMessageSender("KAFKA", cfg, "http://localhost:8080", http.DefaultClient)

	//assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	kafkaSender, ok := messageSender.(*BotKafkaSender)
	if !ok {
		t.Errorf("unexpected sender type: got %T, want *BotKafkaSender", messageSender)
		return
	}

	if err := kafkaSender.Close(); err != nil {
		t.Errorf("close kafka sender: %v", err)
	}
}

func TestNewMessageSender_UnknownTransport_ReturnsError(t *testing.T) {
	//arrange
	cfg := &config.KafkaConfig{
		Brokers:      []string{"localhost:9092"},
		UpdatesTopic: "link-updates",
	}

	//act
	messageSender, err := NewMessageSender("GRPC", cfg, "http://localhost:8080", http.DefaultClient)

	//assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), `unknown updates transport: "GRPC"`) {
		t.Errorf("unexpected error: got %q", err.Error())
	}

	if messageSender != nil {
		t.Errorf("expected nil sender, got %T", messageSender)
	}
}
