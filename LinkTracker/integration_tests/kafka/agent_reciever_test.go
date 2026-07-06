package kafkatest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/integration_tests/helpers"
	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	appservice "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	agentkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/agent"
	botkafka "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/receiver"
	infrasender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/summarizer"
)

func TestAgentKafkaConsumer_ValidRawUpdateEvents_SendsProcessedUpdate(t *testing.T) {
	//arrange
	rawTopic := helpers.UniqueTopic(t, "link-updates-raw")
	processedTopic := helpers.UniqueTopic(t, "link-updates-processed")
	dlqTopic := helpers.UniqueTopic(t, "link-tracker-dlq")

	helpers.CreateTopic(t, kafkaBrokers, rawTopic)
	helpers.CreateTopic(t, kafkaBrokers, processedTopic)
	helpers.CreateTopic(t, kafkaBrokers, dlqTopic)

	kafkaCfg := config.KafkaConfig{
		Brokers: kafkaBrokers,

		ProcessedUpdatesTopic:         processedTopic,
		ProcessedUpdatesConsumerGroup: helpers.UniqueTopic(t, "bot-group"),

		RawUpdatesTopic:         rawTopic,
		RawUpdatesConsumerGroup: helpers.UniqueTopic(t, "agent-group"),

		DLQTopic:               dlqTopic,
		ConsumerMaxAttempts:    3,
		ConsumerBaseRetryDelay: 200 * time.Millisecond,
		ConsumerMaxRetryDelay:  5 * time.Second,
	}

	processingService := newTestProcessingService()
	processedSender := infrasender.NewBotKafkaSender(kafkaBrokers, processedTopic)
	defer func() {
		if err := processedSender.Close(); err != nil {
			t.Errorf("close processed sender: %v", err)
		}
	}()

	agentReceiver := receiver.NewAgentKafkaReceiver(
		kafkaCfg,
		processingService,
		processedSender,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentReceiver.Start(ctx, testLogger(), cancel)
	defer shutdownAgentReceiver(t, agentReceiver, cancel)

	rawMsg := appsender.RawUpdateEvents{
		URL: "https://github.com/example/repo",
		Events: []appsender.Event{
			{
				Source:       string(schedulerlink.TypeGitHub),
				Type:         string(update.GitHubEventIssue),
				Title:        "Issue title",
				Author:       "alice",
				Preview:      "Issue preview with enough length",
				CreationTime: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		TgChatIDs: []int64{1001},
	}

	//act
	writeMessage(t, rawTopic, agentkafka.Message{
		Kind:         agentkafka.MessageKindUpdateEvents,
		UpdateEvents: rawMsg,
	})

	processed := readProcessedUpdateByKey(t, processedTopic, "1")

	//assert
	if processed.URL != rawMsg.URL {
		t.Errorf("unexpected processed url: got %q, want %q", processed.URL, rawMsg.URL)
	}

	if len(processed.TgChatIDs) != 1 || processed.TgChatIDs[0] != 1001 {
		t.Errorf("unexpected chat ids: got %+v, want [1001]", processed.TgChatIDs)
	}

	expectedPrefix := fmt.Sprintf("Появилось обновление по ссылке: %s", rawMsg.URL)
	if !strings.Contains(processed.Description, expectedPrefix) {
		t.Errorf("expected processed description to contain %q, got %q", expectedPrefix, processed.Description)
	}

	if !strings.Contains(processed.Description, "Новый Issue") {
		t.Errorf("expected processed description to contain github event label, got %q", processed.Description)
	}

	if !strings.Contains(processed.Description, "Issue title") {
		t.Errorf("expected processed description to contain event title, got %q", processed.Description)
	}

	if !strings.Contains(processed.Description, "alice") {
		t.Errorf("expected processed description to contain author, got %q", processed.Description)
	}
}

func TestAgentKafkaConsumer_InvalidMessage_DoesNotCrash(t *testing.T) {
	//arrange
	rawTopic := helpers.UniqueTopic(t, "link-updates-raw")
	processedTopic := helpers.UniqueTopic(t, "link-updates-processed")
	dlqTopic := helpers.UniqueTopic(t, "link-tracker-dlq")

	helpers.CreateTopic(t, kafkaBrokers, rawTopic)
	helpers.CreateTopic(t, kafkaBrokers, processedTopic)
	helpers.CreateTopic(t, kafkaBrokers, dlqTopic)

	kafkaCfg := config.KafkaConfig{
		Brokers: kafkaBrokers,

		ProcessedUpdatesTopic:         processedTopic,
		ProcessedUpdatesConsumerGroup: helpers.UniqueTopic(t, "bot-group"),

		RawUpdatesTopic:         rawTopic,
		RawUpdatesConsumerGroup: helpers.UniqueTopic(t, "agent-group"),

		DLQTopic:               dlqTopic,
		ConsumerMaxAttempts:    3,
		ConsumerBaseRetryDelay: 200 * time.Millisecond,
		ConsumerMaxRetryDelay:  5 * time.Second,
	}

	processingService := newTestProcessingService()
	processedSender := infrasender.NewBotKafkaSender(kafkaBrokers, processedTopic)
	defer func() {
		if err := processedSender.Close(); err != nil {
			t.Errorf("close processed sender: %v", err)
		}
	}()

	agentReceiver := receiver.NewAgentKafkaReceiver(
		kafkaCfg,
		processingService,
		processedSender,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentReceiver.Start(ctx, testLogger(), cancel)
	defer shutdownAgentReceiver(t, agentReceiver, cancel)

	validRawMsg := appsender.RawUpdateEvents{
		URL: "https://github.com/example/repo",
		Events: []appsender.Event{
			{
				Source:       string(schedulerlink.TypeGitHub),
				Type:         string(update.GitHubEventIssue),
				Title:        "Issue title",
				Author:       "alice",
				Preview:      "Issue preview with enough length",
				CreationTime: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		TgChatIDs: []int64{1001},
	}

	//act
	helpers.WriteMessage(t, kafkaBrokers, rawTopic, kafka.Message{
		Key:   []byte("bad-message"),
		Value: []byte("{not-valid-json"),
	})

	writeMessage(t, rawTopic, agentkafka.Message{
		Kind:         agentkafka.MessageKindUpdateEvents,
		UpdateEvents: validRawMsg,
	})

	processed := readProcessedUpdateByKey(t, processedTopic, "1")

	//assert
	if processed.URL != validRawMsg.URL {
		t.Errorf("unexpected processed url: got %q, want %q", processed.URL, validRawMsg.URL)
	}

	if len(processed.TgChatIDs) != 1 || processed.TgChatIDs[0] != 1001 {
		t.Errorf("unexpected chat ids: got %+v, want [1001]", processed.TgChatIDs)
	}

	if !strings.Contains(processed.Description, "Issue title") {
		t.Errorf("expected processed description to contain valid event title, got %q", processed.Description)
	}
}

func newTestProcessingService() *appservice.UpdatesProcessingService {
	agentCfg := config.AgentConfig{
		StopWords:        []string{"spam"},
		ExcludedAuthors:  []string{"bot-user"},
		MinLength:        5,
		SummaryThreshold: 1000,
	}

	filter := updates.NewFilter(&agentCfg)
	summarizer := summarizer.NewStubSummarizer(agentCfg.SummaryThreshold)

	return appservice.NewUpdatesProcessingService(
		filter,
		summarizer,
		[]appservice.Formatter{
			updates.GitHubFormatter{},
			updates.StackOverflowFormatter{},
		},
	)
}

func writeMessage(t *testing.T, topic string, msg agentkafka.Message) {
	t.Helper()

	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal raw envelope: %v", err)
	}

	helpers.WriteMessage(t, kafkaBrokers, topic, kafka.Message{
		Key:   []byte("raw-update"),
		Value: payload,
	})
}

func readProcessedUpdateByKey(t *testing.T, topic string, key string) botkafka.LinkUpdate {
	t.Helper()

	msg := helpers.ReadMessageByKey(t, kafkaBrokers, topic, key)

	var updateMsg botkafka.LinkUpdate
	if err := json.Unmarshal(msg.Value, &updateMsg); err != nil {
		t.Fatalf("decode processed update: %v", err)
	}

	return updateMsg
}

func shutdownAgentReceiver(t *testing.T, agentReceiver interface {
	Shutdown() error
}, cancel context.CancelFunc) {
	t.Helper()

	cancel()

	if err := agentReceiver.Shutdown(); err != nil {
		t.Errorf("shutdown ai agent receiver: %v", err)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
