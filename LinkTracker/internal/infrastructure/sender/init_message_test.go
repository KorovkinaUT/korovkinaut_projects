package sender

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func TestNewMessageSender_HTTPTransport_ReturnsFallbackSender(t *testing.T) {
	//arrange
	kafkaCfg := &config.KafkaConfig{
		Brokers:      []string{"localhost:9092"},
		ProcessedUpdatesTopic: "link-updates",
	}
	retryCfg := &config.RetryConfig{
		Attempts:          3,
		Delay:             200 * time.Millisecond,
		RetryableStatuses: []int{500, 502, 503, 504},
	}
	cbCfg := &config.CircuitBreakerConfig{
		SlidingWindowSize:       10,
		FailureRateThreshold:    50,
		CallsInHalfOpen:         5,
		WaitDurationInOpenState: time.Second,
	}

	//act
	messageSender, err := NewMessageSender(
		"HTTP",
		"http://localhost:8080",
		http.DefaultClient,
		kafkaCfg,
		retryCfg,
		cbCfg,
		nil,
	)

	//assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := messageSender.(*BotFallbackSender); !ok {
		t.Errorf("unexpected sender type: got %T, want *FallbackSender", messageSender)
	}
}

func TestNewMessageSender_KafkaTransport_ReturnsKafkaSender(t *testing.T) {
	//arrange
	kafkaCfg := &config.KafkaConfig{
		Brokers:      []string{"localhost:9092"},
		ProcessedUpdatesTopic: "link-updates",
	}
	retryCfg := &config.RetryConfig{
		Attempts:          3,
		Delay:             200 * time.Millisecond,
		RetryableStatuses: []int{500, 502, 503, 504},
	}
	cbCfg := &config.CircuitBreakerConfig{
		SlidingWindowSize:       10,
		FailureRateThreshold:    50,
		CallsInHalfOpen:         5,
		WaitDurationInOpenState: time.Second,
	}

	//act
	messageSender, err := NewMessageSender(
		"KAFKA",
		"http://localhost:8080",
		http.DefaultClient,
		kafkaCfg,
		retryCfg,
		cbCfg,
		nil,
	)

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
	kafkaCfg := &config.KafkaConfig{
		Brokers:      []string{"localhost:9092"},
		ProcessedUpdatesTopic: "link-updates",
	}
	retryCfg := &config.RetryConfig{
		Attempts:          3,
		Delay:             200 * time.Millisecond,
		RetryableStatuses: []int{500, 502, 503, 504},
	}
	cbCfg := &config.CircuitBreakerConfig{
		SlidingWindowSize:       10,
		FailureRateThreshold:    50,
		CallsInHalfOpen:         5,
		WaitDurationInOpenState: time.Second,
	}

	//act
	messageSender, err := NewMessageSender(
		"GRPC",
		"http://localhost:8080",
		http.DefaultClient,
		kafkaCfg,
		retryCfg,
		cbCfg,
		nil,
	)

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
