package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type KafkaConfig struct {
	Brokers              []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`

	ProcessedUpdatesTopic         string   `envconfig:"KAFKA_PROCESSED_UPDATES_TOPIC" default:"link-updates-processed"`
	ProcessedUpdatesConsumerGroup string   `envconfig:"KAFKA_PROCESSED_UPDATES_GROUP" default:"bot"`
	RawUpdatesTopic         string `envconfig:"KAFKA_RAW_UPDATES_TOPIC" default:"link-updates-raw"`
	RawUpdatesConsumerGroup string `envconfig:"KAFKA_RAW_UPDATES_GROUP" default:"agent"`

	DLQTopic               string        `envconfig:"KAFKA_DLQ_TOPIC" default:"dead-letter-queue"`
	ConsumerMaxAttempts    int           `envconfig:"KAFKA_CONSUMER_MAX_ATTEMPTS" default:"3"`
	ConsumerBaseRetryDelay time.Duration `envconfig:"KAFKA_CONSUMER_BASE_RETRY_DELAY" default:"200ms"`
	ConsumerMaxRetryDelay  time.Duration `envconfig:"KAFKA_CONSUMER_MAX_RETRY_DELAY" default:"5s"`
}

func LoadKafkaConfig() (*KafkaConfig, error) {
	var cfg KafkaConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	if cfg.ConsumerMaxAttempts < 1 {
		cfg.ConsumerMaxAttempts = 3
	}
	if cfg.ConsumerBaseRetryDelay < 0 {
		cfg.ConsumerBaseRetryDelay = 200 * time.Millisecond
	}
	if cfg.ConsumerMaxRetryDelay <= 0 {
		cfg.ConsumerMaxRetryDelay = 5 * time.Second
	}

	return &cfg, nil
}
