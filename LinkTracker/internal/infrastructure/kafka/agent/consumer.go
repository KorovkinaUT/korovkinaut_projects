package agentkafka

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	kafkainfra "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/kafka"
)

var ErrBadMessage = errors.New("bad kafka message")

type ProcessingService interface {
	ProcessUpdateEvents(ctx context.Context, msg *sender.RawUpdateEvents) (*sender.UpdateMessage, error)
	ProcessProblems(ctx context.Context, problems []sender.Problem) ([]sender.ProblemsMessage, error)
}

type MessageKind string

const (
	MessageKindUpdateEvents MessageKind = "UPDATE_EVENTS"
	MessageKindProblems     MessageKind = "PROBLEMS"
)

type Message struct {
	Kind         MessageKind            `json:"kind"`
	UpdateEvents sender.RawUpdateEvents `json:"updateEvents,omitempty"`
	Problems     []sender.Problem       `json:"problems,omitempty"`
}

// Reads raw updates from scrapper, processes them and sends to bot
type Consumer struct {
	reader        *kafka.Reader
	dlqWriter     *kafka.Writer
	dlqTopic      string
	maxAttempts   int
	retryDelay    time.Duration
	maxDelay      time.Duration
	processor     ProcessingService
	messageSender sender.MessageSender
}

func NewConsumer(
	cfg config.KafkaConfig,
	processor ProcessingService,
	messageSender sender.MessageSender,
) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: cfg.Brokers,
			Topic:   cfg.RawUpdatesTopic,
			GroupID: cfg.RawUpdatesConsumerGroup,
		}),
		dlqWriter: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Brokers...),
			Topic:    cfg.DLQTopic,
			Balancer: &kafka.Hash{},
		},
		dlqTopic:      cfg.DLQTopic,
		maxAttempts:   cfg.ConsumerMaxAttempts,
		retryDelay:    cfg.ConsumerBaseRetryDelay,
		maxDelay:      cfg.ConsumerMaxRetryDelay,
		processor:     processor,
		messageSender: messageSender,
	}
}

func (c *Consumer) Start(ctx context.Context, logger *slog.Logger) error {
	logger.Info("starting agent kafka consumer")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				logger.Info("agent kafka consumer stopped")
				return nil
			}

			return fmt.Errorf("fetch kafka message: %w", err)
		}

		if err := c.handleMessageWithRetry(ctx, logger, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("agent kafka consumer stopped")
				return nil
			}

			return err
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("agent kafka consumer stopped")
				return nil
			}

			return fmt.Errorf("commit kafka message: %w", err)
		}
	}
}

func (c *Consumer) Close() error {
	var result error

	if err := c.reader.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("close kafka reader: %w", err))
	}

	if err := c.dlqWriter.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("close kafka dlq writer: %w", err))
	}

	return result
}

func (c *Consumer) handleMessageWithRetry(
	ctx context.Context,
	logger *slog.Logger,
	kafkaMsg kafka.Message,
) error {
	var err error

	for attempt := 0; attempt < c.maxAttempts; attempt++ {
		err = c.handleMessage(ctx, kafkaMsg)
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) {
			return err
		}

		if errors.Is(err, ErrBadMessage) {
			logger.Error(
				"bad kafka message, sending to dlq",
				slog.String("topic", kafkaMsg.Topic),
				slog.Int("partition", kafkaMsg.Partition),
				slog.Int64("offset", kafkaMsg.Offset),
				slog.String("error", err.Error()),
			)

			if dlqErr := c.sendToDLQ(ctx, kafkaMsg, err); dlqErr != nil {
				return dlqErr
			}

			return nil
		}

		if attempt != c.maxAttempts-1 {
			timer := time.NewTimer(min(c.maxDelay, c.retryDelay*time.Duration(1<<attempt)))

			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	logger.Error(
		"kafka message attempts exhausted, sending to dlq",
		slog.String("topic", kafkaMsg.Topic),
		slog.Int("partition", kafkaMsg.Partition),
		slog.Int64("offset", kafkaMsg.Offset),
		slog.Int("max_attempts", c.maxAttempts),
		slog.String("error", err.Error()),
	)

	if dlqErr := c.sendToDLQ(ctx, kafkaMsg, err); dlqErr != nil {
		return dlqErr
	}

	return nil
}

func (c *Consumer) sendToDLQ(ctx context.Context, kafkaMsg kafka.Message, cause error) error {
	msg := kafkainfra.DeadLetterMessage{
		OriginalTopic:     kafkaMsg.Topic,
		OriginalPartition: kafkaMsg.Partition,
		OriginalOffset:    kafkaMsg.Offset,
		Key:               string(kafkaMsg.Key),
		Value:             string(kafkaMsg.Value),
		Error:             cause.Error(),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal dlq message: %w", err)
	}

	err = c.dlqWriter.WriteMessages(ctx, kafka.Message{
		Key:   kafkaMsg.Key,
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("write message to kafka dlq topic %q: %w", c.dlqTopic, err)
	}

	return nil
}

func (c *Consumer) handleMessage(ctx context.Context, kafkaMsg kafka.Message) error {
	msg, err := decodeMessage(kafkaMsg.Value)
	if err != nil {
		return err
	}

	if err := validateMessage(msg); err != nil {
		return err
	}

	switch msg.Kind {
	case MessageKindUpdateEvents:
		updateMsg, err := c.processor.ProcessUpdateEvents(ctx, &msg.UpdateEvents)
		if err != nil {
			return fmt.Errorf("process update events: %w", err)
		}

		if updateMsg == nil {
			return nil
		}

		if err := c.messageSender.SendUpdate(ctx, *updateMsg); err != nil {
			return fmt.Errorf("send processed update: %w", err)
		}

		return nil

	case MessageKindProblems:
		problemMessages, err := c.processor.ProcessProblems(ctx, msg.Problems)
		if err != nil {
			return fmt.Errorf("process problems: %w", err)
		}

		for _, problemMsg := range problemMessages {
			if err := c.messageSender.SendProblems(ctx, problemMsg); err != nil {
				return fmt.Errorf("send processed problems: %w", err)
			}
		}

		return nil

	default:
		return fmt.Errorf("%w: unsupported raw message kind %q", ErrBadMessage, msg.Kind)
	}
}

func decodeMessage(value []byte) (*Message, error) {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()

	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		return nil, fmt.Errorf("%w: decode raw message envelope: %v", ErrBadMessage, err)
	}

	return &msg, nil
}

func validateMessage(msg *Message) error {
	switch msg.Kind {
	case MessageKindUpdateEvents:
		return validateRawUpdateEvents(msg.UpdateEvents)
	case MessageKindProblems:
		return validateRawProblems(msg.Problems)
	default:
		return fmt.Errorf("%w: unsupported raw message kind %q", ErrBadMessage, msg.Kind)
	}
}

func validateRawUpdateEvents(msg sender.RawUpdateEvents) error {
	if msg.URL == "" {
		return fmt.Errorf("%w: url must be present", ErrBadMessage)
	}

	if msg.Events == nil {
		return fmt.Errorf("%w: events must be present", ErrBadMessage)
	}

	if msg.TgChatIDs == nil {
		return fmt.Errorf("%w: tgChatIds must be present", ErrBadMessage)
	}

	for i, event := range msg.Events {
		if err := validateEvent(event); err != nil {
			return fmt.Errorf("%w: event %d: %v", ErrBadMessage, i, err)
		}
	}

	return nil
}

func validateEvent(event sender.Event) error {
	if event.Source == "" {
		return errors.New("source must be present")
	}
	if event.Type == "" {
		return errors.New("type must be present")
	}
	if event.Title == "" {
		return errors.New("title must be present")
	}
	if event.Author == "" {
		return errors.New("author must be present")
	}
	if event.Preview == "" {
		return errors.New("preview must be present")
	}
	if event.CreationTime.IsZero() {
		return errors.New("creationTime must be present")
	}

	return nil
}

func validateRawProblems(problems []sender.Problem) error {
	for i, problem := range problems {
		if err := validateProblem(problem); err != nil {
			return fmt.Errorf("%w: problem %d: %v", ErrBadMessage, i, err)
		}
	}

	return nil
}

func validateProblem(problem sender.Problem) error {
	if problem.URL == "" {
		return errors.New("url must be present")
	}
	if problem.Message == "" {
		return errors.New("message must be present")
	}
	if problem.ChatIDs == nil {
		return errors.New("chatIds must be present")
	}

	return nil
}
