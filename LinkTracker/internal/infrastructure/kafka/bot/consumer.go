package botkafka

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// to send to DLQ immediately
var ErrBadMessage = errors.New("bad kafka message")

type LinkUpdate struct {
	ID          int64   `json:"id"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	TgChatIDs   []int64 `json:"tgChatIds"`
}

// for DLQ
type DeadLetterMessage struct {
	OriginalTopic     string `json:"originalTopic"`
	OriginalPartition int    `json:"originalPartition"`
	OriginalOffset    int64  `json:"originalOffset"`
	Key               string `json:"key"`
	Value             string `json:"value"`
	Error             string `json:"error"`
}

// Consumer reads link update notifications from Kafka and sends them to Telegram chats
type Consumer struct {
	reader      *kafka.Reader
	dlqWriter   *kafka.Writer
	dlqTopic    string
	maxAttempts int
	sendMessage func(chatID int64, text string) error
}

func NewConsumer(
	brokers []string,
	topic string,
	groupID string,
	dlqTopic string,
	maxAttempts int,
	sendMessage func(chatID int64, text string) error,
) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		dlqWriter: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    dlqTopic,
			Balancer: &kafka.Hash{},
		},
		dlqTopic:    dlqTopic,
		maxAttempts: maxAttempts,
		sendMessage: sendMessage,
	}
}

func (c *Consumer) Start(ctx context.Context, logger *slog.Logger) error {
	logger.Info("starting kafka consumer")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				logger.Info("kafka consumer stopped")
				return nil
			}

			return fmt.Errorf("fetch kafka message: %w", err)
		}

		if err := c.handleMessageWithRetry(ctx, logger, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("kafka consumer stopped")
				return nil
			}

			return err
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("kafka consumer stopped")
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

func (c *Consumer) handleMessageWithRetry(ctx context.Context, logger *slog.Logger, kafkaMsg kafka.Message) error {
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
	msg := DeadLetterMessage{
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
	msg, err := decodeLinkUpdate(kafkaMsg.Value)
	if err != nil {
		return err
	}

	if err := validateLinkUpdate(msg); err != nil {
		return err
	}

	for _, chatID := range msg.TgChatIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.sendMessage(chatID, msg.Description); err != nil {
			return fmt.Errorf("send telegram message to chat %d: %w", chatID, err)
		}
	}

	return nil
}

func decodeLinkUpdate(value []byte) (LinkUpdate, error) {
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()

	var msg LinkUpdate
	if err := decoder.Decode(&msg); err != nil {
		return LinkUpdate{}, fmt.Errorf("%w: decode link update: %v", ErrBadMessage, err)
	}

	return msg, nil
}

func validateLinkUpdate(msg LinkUpdate) error {
	if msg.ID == 0 {
		return fmt.Errorf("%w: id must be present", ErrBadMessage)
	}

	if msg.URL == "" {
		return fmt.Errorf("%w: url must be present", ErrBadMessage)
	}

	if msg.Description == "" {
		return fmt.Errorf("%w: description must be present", ErrBadMessage)
	}

	if msg.TgChatIDs == nil {
		return fmt.Errorf("%w: tgChatIds must be present", ErrBadMessage)
	}

	return nil
}
