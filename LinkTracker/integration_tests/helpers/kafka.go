package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func StartKafka(ctx context.Context) ([]string, func(context.Context) error, error) {
	container, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.8.0")
	if err != nil {
		return nil, nil, fmt.Errorf("start kafka container: %w", err)
	}

	brokers, err := container.Brokers(ctx)
	if err != nil {
		_ = container.Terminate(context.Background())
		return nil, nil, fmt.Errorf("get kafka brokers: %w", err)
	}

	terminate := func(ctx context.Context) error {
		return container.Terminate(ctx)
	}

	return brokers, terminate, nil
}

func UniqueTopic(t *testing.T, prefix string) string {
	t.Helper()

	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func CreateTopic(t *testing.T, brokers []string, topic string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		t.Fatalf("dial kafka broker: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close kafka connection: %v", err)
		}
	}()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		t.Fatalf("create topic %q: %v", topic, err)
	}

	WaitForTopic(t, brokers, topic)
}

func WaitForTopic(t *testing.T, brokers []string, topic string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		err := checkTopicReady(brokers, topic)
		if err == nil {
			// Kafka может уже вернуть metadata, но writer иногда ещё не успевает
			// увидеть topic сразу после создания. Небольшая пауза делает тесты стабильнее.
			time.Sleep(500 * time.Millisecond)
			return
		}

		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("topic %q is not ready: %v", topic, lastErr)
}

func ReadMessage(t *testing.T, brokers []string, topic string) kafka.Message {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   brokers,
		Topic:     topic,
		Partition: 0,
		MaxWait:   time.Second,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			t.Errorf("close kafka reader: %v", err)
		}
	}()

	if err := reader.SetOffset(kafka.FirstOffset); err != nil {
		t.Fatalf("set kafka reader offset: %v", err)
	}

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("read kafka message: %v", err)
	}

	return msg
}

func RetryUntilSuccess(t *testing.T, timeout time.Duration, action func() error) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if err := action(); err == nil {
			return
		} else {
			lastErr = err
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("action did not succeed within %s: %v", timeout, lastErr)
}

func WriteMessage(t *testing.T, brokers []string, topic string, msg kafka.Message) {
	t.Helper()

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			t.Errorf("close kafka writer: %v", err)
		}
	}()

	RetryUntilSuccess(t, 30*time.Second, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		return writer.WriteMessages(ctx, msg)
	})
}

func AssertEventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("condition was not met within %s", timeout)
}

func checkTopicReady(brokers []string, topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka broker: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		return fmt.Errorf("read topic partitions: %w", err)
	}

	for _, partition := range partitions {
		if partition.Topic == topic && partition.ID == 0 && partition.Leader.Host != "" {
			return nil
		}
	}

	return fmt.Errorf("topic %q partition 0 with leader not found", topic)
}
