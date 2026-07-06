package kafkatest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	botkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/bot"
	infrasender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
)

func TestBotKafkaSender_SendUpdate_WritesExpectedMessageToKafka(t *testing.T) {
	//arrange
	topic := helpers.UniqueTopic(t, "link-updates-test")
	helpers.CreateTopic(t, kafkaBrokers, topic)

	kafkaSender := infrasender.NewBotKafkaSender(kafkaBrokers, topic)
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

func TestBotSender_HTTPUnavailable_SendsUpdateToKafka(t *testing.T) {
	//arrange
	topic := helpers.UniqueTopic(t, "fallback-updates")
	helpers.CreateTopic(t, kafkaBrokers, topic)
	helpers.WaitForTopicWritable(t, kafkaBrokers, topic)

	var httpRequestsCount int32
	botHTTPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&httpRequestsCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer botHTTPServer.Close()

	kafkaCfg := &config.KafkaConfig{
		Brokers:      kafkaBrokers,
		ProcessedUpdatesTopic: topic,
	}
	retryCfg := &config.RetryConfig{
		Attempts:          1,
		Delay:             time.Millisecond,
		RetryableStatuses: []int{http.StatusInternalServerError},
	}
	cbCfg := &config.CircuitBreakerConfig{
		SlidingWindowSize:       100,
		FailureRateThreshold:    100,
		CallsInHalfOpen:         1,
		WaitDurationInOpenState: time.Second,
	}

	messageSender, err := infrasender.NewMessageSender(
		"HTTP",
		botHTTPServer.URL,
		botHTTPServer.Client(),
		kafkaCfg,
		retryCfg,
		cbCfg,
		nil,
	)
	if err != nil {
		t.Fatalf("create message sender: %v", err)
	}

	if closer, ok := messageSender.(interface{ Close() error }); ok {
		defer func() {
			if err := closer.Close(); err != nil {
				t.Errorf("close message sender: %v", err)
			}
		}()
	}

	update := appsender.UpdateMessage{
		ID:          123,
		URL:         "https://github.com/user/repo",
		Description: "test update",
		TgChatIDs:   []int64{1, 2},
	}

	//act
	err = messageSender.SendUpdate(context.Background(), update)

	//assert
	if err != nil {
		t.Fatalf("send update with fallback: %v", err)
	}

	if got := atomic.LoadInt32(&httpRequestsCount); got != 1 {
		t.Errorf("unexpected http requests count: got %d, want 1", got)
	}

	expectedKey := strconv.FormatInt(update.ID, 10)
	kafkaMessage := helpers.ReadMessageByKey(t, kafkaBrokers, topic, expectedKey)

	var received botkafka.LinkUpdate
	if err := json.Unmarshal(kafkaMessage.Value, &received); err != nil {
		t.Fatalf("unmarshal kafka message: %v", err)
	}

	if received.ID != update.ID {
		t.Errorf("unexpected update id: got %d, want %d", received.ID, update.ID)
	}

	if received.URL != update.URL {
		t.Errorf("unexpected update url: got %q, want %q", received.URL, update.URL)
	}

	if len(received.TgChatIDs) != len(update.TgChatIDs) {
		t.Errorf("unexpected chat ids count: got %d, want %d", len(received.TgChatIDs), len(update.TgChatIDs))
	}

	if len(received.TgChatIDs) == len(update.TgChatIDs) {
		for i := range update.TgChatIDs {
			if received.TgChatIDs[i] != update.TgChatIDs[i] {
				t.Errorf("unexpected chat id at index %d: got %d, want %d", i, received.TgChatIDs[i], update.TgChatIDs[i])
			}
		}
	}

	if !strings.Contains(received.Description, update.URL) {
		t.Errorf("description must contain update url: got %q", received.Description)
	}

	if !strings.Contains(received.Description, update.Description) {
		t.Errorf("description must contain original update description: got %q", received.Description)
	}
}
