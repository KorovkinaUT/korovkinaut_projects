package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type ScrapperConfig struct {
	SchedulerInterval time.Duration `envconfig:"SCHEDULER_INTERVAL" default:"1m"`
	BatchSize         int64         `envconfig:"BATCH_SIZE" default:"100"`
	WorkersCount      int           `envconfig:"WORKERS_COUNT" default:"4"`

	UpdatesTransport string `envconfig:"UPDATES_TRANSPORT" default:"KAFKA"`

	BotHost      string `envconfig:"BOT_HOST" default:"localhost"`
	BotPort      int    `envconfig:"BOT_PORT" default:"8080"`
	ScrapperHost string `envconfig:"SCRAPPER_HOST" default:"localhost"`
	ScrapperPort int    `envconfig:"SCRAPPER_PORT" default:"8081"`

	HttpTimeout     time.Duration `envconfig:"HTTP_TIMEOUT" default:"5s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`

	GithubBaseURL        string `envconfig:"GITHUB_BASE_URL" default:"https://api.github.com"`
	StackOverflowBaseURL string `envconfig:"STACKOVERFLOW_BASE_URL" default:"https://api.stackexchange.com/2.3"`
}

func (c *ScrapperConfig) ScrapperAddress() string {
	return fmt.Sprintf(":%d", c.ScrapperPort)
}

func (c *ScrapperConfig) BotBaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.BotHost, c.BotPort)
}

func LoadScrapperConfig() (*ScrapperConfig, error) {
	var cfg ScrapperConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	if strings.ToUpper(cfg.UpdatesTransport) != "HTTP" && strings.ToUpper(cfg.UpdatesTransport) != "KAFKA" {
		return nil, fmt.Errorf(
			"unsupported UPDATES_TRANSPORT %q: expected one of [HTTP, KAFKA]",
			cfg.UpdatesTransport,
		)
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.WorkersCount <= 0 {
		cfg.WorkersCount = 1
	}
	if cfg.WorkersCount > int(cfg.BatchSize) {
		cfg.WorkersCount = int(cfg.BatchSize)
	}
	if cfg.HttpTimeout <= 0 {
		cfg.HttpTimeout = 5 * time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}
	if cfg.SchedulerInterval <= 0 {
		cfg.SchedulerInterval = time.Minute
	}

	return &cfg, nil
}
