package kafkatest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	botkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/receiver"

	"github.com/segmentio/kafka-go"
)

func TestBotKafkaReceiver_ValidMessage_HandlesMessage(t *testing.T) {
	//arrange
	topic := helpers.UniqueTopic(t, "link-updates-valid")
	dlqTopic := helpers.UniqueTopic(t, "link-tracker-dlq")
	helpers.CreateTopic(t, kafkaBrokers, topic)
	helpers.CreateTopic(t, kafkaBrokers, dlqTopic)

	var (
		mu       sync.Mutex
		chatIDs  []int64
		messages []string
	)

	sendMessage := func(chatID int64, text string) error {
		mu.Lock()
		defer mu.Unlock()

		chatIDs = append(chatIDs, chatID)
		messages = append(messages, text)

		return nil
	}

	kafkaReceiver := receiver.NewKafkaReceiver(
		kafkaBrokers,
		topic,
		helpers.UniqueTopic(t, "consumer-group"),
		dlqTopic,
		3,
		sendMessage,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kafkaReceiver.Start(ctx, nilLogger(), cancel)

	defer func() {
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := kafkaReceiver.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown kafka receiver: %v", err)
		}
	}()

	update := botkafka.LinkUpdate{
		ID:          100,
		URL:         "https://github.com/example/repo",
		Description: "Появилось обновление",
		TgChatIDs:   []int64{1001, 1002},
	}

	payload, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("marshal update: %v", err)
	}

	//act
	helpers.WriteMessage(t, kafkaBrokers, topic, kafka.Message{
		Key:   []byte("100"),
		Value: payload,
	})

	helpers.AssertEventually(t, 10*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return len(chatIDs) == 2 && len(messages) == 2
	})

	//assert
	mu.Lock()
	defer mu.Unlock()

	if chatIDs[0] != 1001 {
		t.Errorf("unexpected first chat id: got %d, want %d", chatIDs[0], 1001)
	}

	if chatIDs[1] != 1002 {
		t.Errorf("unexpected second chat id: got %d, want %d", chatIDs[1], 1002)
	}

	for i, message := range messages {
		if message != update.Description {
			t.Errorf("unexpected message[%d]: got %q, want %q", i, message, update.Description)
		}
	}
}

func TestBotKafkaReceiver_InvalidMessage_SendsMessageToDLQ(t *testing.T) {
	tests := []struct {
		name              string
		value             string
		expectedErrorPart string
	}{
		{
			name:              "invalid json format",
			value:             `{"id":100,"url":"https://github.com/example/repo","description":"test","tgChatIds":[1001]`,
			expectedErrorPart: "decode link update",
		},
		{
			name:              "invalid message data",
			value:             `{"id":100,"url":"","description":"test","tgChatIds":[1001]}`,
			expectedErrorPart: "url must be present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//arrange
			topic := helpers.UniqueTopic(t, "link-updates-invalid")
			dlqTopic := helpers.UniqueTopic(t, "link-tracker-dlq")
			helpers.CreateTopic(t, kafkaBrokers, topic)
			helpers.CreateTopic(t, kafkaBrokers, dlqTopic)

			var sendMessageCalls atomic.Int64

			sendMessage := func(chatID int64, text string) error {
				sendMessageCalls.Add(1)
				return nil
			}

			kafkaReceiver := receiver.NewKafkaReceiver(
				kafkaBrokers,
				topic,
				helpers.UniqueTopic(t, "consumer-group"),
				dlqTopic,
				3,
				sendMessage,
			)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			kafkaReceiver.Start(ctx, nilLogger(), cancel)

			defer func() {
				cancel()

				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer shutdownCancel()

				if err := kafkaReceiver.Shutdown(shutdownCtx); err != nil {
					t.Errorf("shutdown kafka receiver: %v", err)
				}
			}()

			//act
			helpers.WriteMessage(t, kafkaBrokers, topic, kafka.Message{
				Key:   []byte("100"),
				Value: []byte(tt.value),
			})

			dlqKafkaMsg := helpers.ReadMessage(t, kafkaBrokers, dlqTopic)

			//assert
			if sendMessageCalls.Load() != 0 {
				t.Errorf("sendMessage was called %d times, want 0", sendMessageCalls.Load())
			}

			if string(dlqKafkaMsg.Key) != "100" {
				t.Errorf("unexpected dlq message key: got %q, want %q", string(dlqKafkaMsg.Key), "100")
			}

			var dlqMsg botkafka.DeadLetterMessage
			if err := json.Unmarshal(dlqKafkaMsg.Value, &dlqMsg); err != nil {
				t.Fatalf("unmarshal dlq message: %v", err)
			}

			if dlqMsg.OriginalTopic != topic {
				t.Errorf("unexpected original topic: got %q, want %q", dlqMsg.OriginalTopic, topic)
			}

			if dlqMsg.OriginalPartition != 0 {
				t.Errorf("unexpected original partition: got %d, want 0", dlqMsg.OriginalPartition)
			}

			if dlqMsg.Key != "100" {
				t.Errorf("unexpected original key: got %q, want %q", dlqMsg.Key, "100")
			}

			if dlqMsg.Value != tt.value {
				t.Errorf("unexpected original value: got %q, want %q", dlqMsg.Value, tt.value)
			}

			if !strings.Contains(dlqMsg.Error, botkafka.ErrBadMessage.Error()) {
				t.Errorf("dlq error should contain %q, got %q", botkafka.ErrBadMessage.Error(), dlqMsg.Error)
			}

			if !strings.Contains(dlqMsg.Error, tt.expectedErrorPart) {
				t.Errorf("dlq error should contain %q, got %q", tt.expectedErrorPart, dlqMsg.Error)
			}
		})
	}
}

func TestBotKafkaReceiver_ProcessingError_RetriesMessage(t *testing.T) {
	t.Run("message is successfully processed after retries", func(t *testing.T) {
		//arrange
		topic := helpers.UniqueTopic(t, "link-updates-retry-success")
		dlqTopic := helpers.UniqueTopic(t, "link-tracker-dlq")
		helpers.CreateTopic(t, kafkaBrokers, topic)
		helpers.CreateTopic(t, kafkaBrokers, dlqTopic)

		const maxAttempts = 3

		var sendMessageCalls atomic.Int64

		sendMessage := func(chatID int64, text string) error {
			call := sendMessageCalls.Add(1)
			if call < maxAttempts {
				return errors.New("temporary processing error")
			}

			return nil
		}

		kafkaReceiver := receiver.NewKafkaReceiver(
			kafkaBrokers,
			topic,
			helpers.UniqueTopic(t, "consumer-group"),
			dlqTopic,
			maxAttempts,
			sendMessage,
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kafkaReceiver.Start(ctx, nilLogger(), cancel)

		defer func() {
			cancel()

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			if err := kafkaReceiver.Shutdown(shutdownCtx); err != nil {
				t.Errorf("shutdown kafka receiver: %v", err)
			}
		}()

		update := botkafka.LinkUpdate{
			ID:          200,
			URL:         "https://github.com/example/repo",
			Description: "Появилось обновление",
			TgChatIDs:   []int64{1001},
		}

		writeLinkUpdate(t, topic, update)

		//act
		helpers.AssertEventually(t, 10*time.Second, func() bool {
			return sendMessageCalls.Load() == maxAttempts
		})

		//assert
		if sendMessageCalls.Load() != maxAttempts {
			t.Errorf("unexpected sendMessage calls: got %d, want %d", sendMessageCalls.Load(), maxAttempts)
		}

		if msg, ok := tryReadMessage(t, dlqTopic, 500*time.Millisecond); ok {
			t.Errorf("unexpected message in dlq: key=%q value=%q", string(msg.Key), string(msg.Value))
		}
	})

	t.Run("message is sent to dlq after attempts exhausted", func(t *testing.T) {
		//arrange
		topic := helpers.UniqueTopic(t, "link-updates-retry-exhausted")
		dlqTopic := helpers.UniqueTopic(t, "link-tracker-dlq")
		helpers.CreateTopic(t, kafkaBrokers, topic)
		helpers.CreateTopic(t, kafkaBrokers, dlqTopic)

		const maxAttempts = 3

		var sendMessageCalls atomic.Int64

		sendMessage := func(chatID int64, text string) error {
			sendMessageCalls.Add(1)
			return errors.New("permanent processing error")
		}

		kafkaReceiver := receiver.NewKafkaReceiver(
			kafkaBrokers,
			topic,
			helpers.UniqueTopic(t, "consumer-group"),
			dlqTopic,
			maxAttempts,
			sendMessage,
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kafkaReceiver.Start(ctx, nilLogger(), cancel)

		defer func() {
			cancel()

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			if err := kafkaReceiver.Shutdown(shutdownCtx); err != nil {
				t.Errorf("shutdown kafka receiver: %v", err)
			}
		}()

		update := botkafka.LinkUpdate{
			ID:          201,
			URL:         "https://github.com/example/repo",
			Description: "Появилось обновление",
			TgChatIDs:   []int64{1001},
		}

		writeLinkUpdate(t, topic, update)

		//act
		dlqKafkaMsg := helpers.ReadMessage(t, kafkaBrokers, dlqTopic)

		//assert
		if sendMessageCalls.Load() != maxAttempts {
			t.Errorf("unexpected sendMessage calls: got %d, want %d", sendMessageCalls.Load(), maxAttempts)
		}

		if string(dlqKafkaMsg.Key) != "201" {
			t.Errorf("unexpected dlq message key: got %q, want %q", string(dlqKafkaMsg.Key), "201")
		}

		var dlqMsg botkafka.DeadLetterMessage
		if err := json.Unmarshal(dlqKafkaMsg.Value, &dlqMsg); err != nil {
			t.Fatalf("unmarshal dlq message: %v", err)
		}

		if dlqMsg.OriginalTopic != topic {
			t.Errorf("unexpected original topic: got %q, want %q", dlqMsg.OriginalTopic, topic)
		}

		if dlqMsg.Key != "201" {
			t.Errorf("unexpected original key: got %q, want %q", dlqMsg.Key, "201")
		}

		if !strings.Contains(dlqMsg.Error, "permanent processing error") {
			t.Errorf("dlq error should contain processing error, got %q", dlqMsg.Error)
		}

		if strings.Contains(dlqMsg.Error, botkafka.ErrBadMessage.Error()) {
			t.Errorf("processing error should not be marked as bad message, got %q", dlqMsg.Error)
		}
	})
}

func nilLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func writeLinkUpdate(t *testing.T, topic string, update botkafka.LinkUpdate) {
	t.Helper()

	payload, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("marshal link update: %v", err)
	}

	helpers.WriteMessage(t, kafkaBrokers, topic, kafka.Message{
		Key:   []byte(strconv.FormatInt(update.ID, 10)),
		Value: payload,
	})
}

func tryReadMessage(t *testing.T, topic string, timeout time.Duration) (kafka.Message, bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   kafkaBrokers,
		Topic:     topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6,
		MaxWait:   100 * time.Millisecond,
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
		if errors.Is(err, context.DeadlineExceeded) {
			return kafka.Message{}, false
		}

		t.Fatalf("read kafka message: %v", err)
	}

	return msg, true
}
