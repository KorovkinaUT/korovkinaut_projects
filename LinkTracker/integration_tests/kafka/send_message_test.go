package kafkatest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/receiver"
	infrasender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
)

func TestSending_ToBotKafkaReceiver_DeliversMessageToUser(t *testing.T) {
	//arrange
	topic := helpers.UniqueTopic(t, "link-updates-full-flow")
	dlqTopic := helpers.UniqueTopic(t, "link-tracker-dlq")
	helpers.CreateTopic(t, kafkaBrokers, topic)
	helpers.CreateTopic(t, kafkaBrokers, dlqTopic)

	const (
		updateID = int64(300)
		chatID   = int64(1001)
		url      = "https://github.com/example/repo"
		text     = "New issue was created"
	)

	var (
		mu              sync.Mutex
		receivedChatIDs []int64
		receivedTexts   []string
	)

	sendMessage := func(chatID int64, text string) error {
		mu.Lock()
		defer mu.Unlock()

		receivedChatIDs = append(receivedChatIDs, chatID)
		receivedTexts = append(receivedTexts, text)

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

	kafkaSender := infrasender.NewKafkaSender(kafkaBrokers, topic)
	defer func() {
		if err := kafkaSender.Close(); err != nil {
			t.Errorf("close kafka sender: %v", err)
		}
	}()

	update := appsender.UpdateMessage{
		ID:          updateID,
		URL:         url,
		Description: text,
		TgChatIDs:   []int64{chatID},
	}

	//act
	helpers.RetryUntilSuccess(t, 30*time.Second, func() error {
		sendCtx, sendCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer sendCancel()

		return kafkaSender.SendUpdate(sendCtx, update)
	})

	helpers.AssertEventually(t, 10*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return len(receivedChatIDs) == 1 && len(receivedTexts) == 1
	})

	//assert
	mu.Lock()
	defer mu.Unlock()

	if receivedChatIDs[0] != chatID {
		t.Errorf("unexpected chat id: got %d, want %d", receivedChatIDs[0], chatID)
	}

	expectedText := fmt.Sprintf(
		"Появилось обновление по ссылке: %s\n\n%s",
		url,
		text,
	)

	if receivedTexts[0] != expectedText {
		t.Errorf("unexpected text: got %q, want %q", receivedTexts[0], expectedText)
	}
}
