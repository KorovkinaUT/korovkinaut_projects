package kafkatest

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	botkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/bot"
	infrasender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
)

func TestBotKafkaSender_SendUpdate_WritesExpectedMessageToKafka(t *testing.T) {
	//arrange
	topic := helpers.UniqueTopic(t, "link-updates-test")
	helpers.CreateTopic(t, kafkaBrokers, topic)

	kafkaSender := infrasender.NewKafkaSender(kafkaBrokers, topic)
	defer func() {
		if err := kafkaSender.Close(); err != nil {
			t.Errorf("close kafka sender: %v", err)
		}
	}()

	msg := appsender.UpdateMessage{
		ID:          42,
		URL:         "https://github.com/example/repo",
		Description: "New issue was created",
		TgChatIDs:   []int64{1001, 1002},
	}

	//act
	helpers.RetryUntilSuccess(t, 30*time.Second, func() error {
		sendCtx, sendCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer sendCancel()

		return kafkaSender.SendUpdate(sendCtx, msg)
	})

	kafkaMsg := helpers.ReadMessage(t, kafkaBrokers, topic)

	//assert
	expectedKey := strconv.FormatInt(msg.ID, 10)
	if string(kafkaMsg.Key) != expectedKey {
		t.Errorf("unexpected kafka message key: got %q, want %q", string(kafkaMsg.Key), expectedKey)
	}

	var actual botkafka.LinkUpdate
	if err := json.Unmarshal(kafkaMsg.Value, &actual); err != nil {
		t.Fatalf("unmarshal kafka message value: %v", err)
	}

	expectedDescription := fmt.Sprintf(
		"Появилось обновление по ссылке: %s\n\n%s",
		msg.URL,
		msg.Description,
	)

	if actual.ID != msg.ID {
		t.Errorf("unexpected id: got %d, want %d", actual.ID, msg.ID)
	}

	if actual.URL != msg.URL {
		t.Errorf("unexpected url: got %q, want %q", actual.URL, msg.URL)
	}

	if actual.Description != expectedDescription {
		t.Errorf("unexpected description: got %q, want %q", actual.Description, expectedDescription)
	}

	if len(actual.TgChatIDs) != len(msg.TgChatIDs) {
		t.Fatalf("unexpected tgChatIds length: got %d, want %d", len(actual.TgChatIDs), len(msg.TgChatIDs))
	}

	for i := range msg.TgChatIDs {
		if actual.TgChatIDs[i] != msg.TgChatIDs[i] {
			t.Errorf("unexpected tgChatIds[%d]: got %d, want %d", i, actual.TgChatIDs[i], msg.TgChatIDs[i])
		}
	}
}
