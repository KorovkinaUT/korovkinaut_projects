package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type KafkaConfig struct {
	Brokers              []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	UpdatesTopic         string   `envconfig:"KAFKA_UPDATES_TOPIC" default:"link-updates"`
	UpdatesConsumerGroup string   `envconfig:"KAFKA_UPDATES_GROUP" default:"bot"`

	DLQTopic            string        `envconfig:"KAFKA_DLQ_TOPIC" default:"dead-letter-queue"`
	ConsumerMaxAttempts int           `envconfig:"KAFKA_CONSUMER_MAX_ATTEMPTS" default:"3"`
	ConsumerRetryDelay  time.Duration `envconfig:"KAFKA_CONSUMER_RETRY_DELAY" default:"200ms"`
}

func LoadKafkaConfig() (*KafkaConfig, error) {
	var cfg KafkaConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	if cfg.ConsumerMaxAttempts < 1 {
		cfg.ConsumerMaxAttempts = 3
	}
	if cfg.ConsumerRetryDelay < 0 {
		cfg.ConsumerRetryDelay = 200 * time.Millisecond
	}

	return &cfg, nil
}
