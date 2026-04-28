package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type BotConfig struct {
	AppTelegramToken   string `envconfig:"APP_TELEGRAM_TOKEN" required:"true"`
	PollTimeoutSeconds int    `envconfig:"POLL_TIMEOUT_SECONDS" default:"30s"`

	UpdatesTransport string `envconfig:"UPDATES_TRANSPORT" default:"KAFKA"`

	BotHost      string `envconfig:"BOT_HOST" default:"localhost"`
	BotPort      int    `envconfig:"BOT_PORT" default:"8080"`
	ScrapperHost string `envconfig:"SCRAPPER_HOST" default:"localhost"`
	ScrapperPort int    `envconfig:"SCRAPPER_PORT" default:"8081"`

	HttpTimeout     time.Duration `envconfig:"HTTP_TIMEOUT" default:"5s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
}

func (c *BotConfig) BotAddress() string {
	return fmt.Sprintf(":%d", c.BotPort)
}

func (c *BotConfig) ScrapperBaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.ScrapperHost, c.ScrapperPort)
}

func LoadBotConfig() (*BotConfig, error) {
	var cfg BotConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	switch strings.ToUpper(cfg.UpdatesTransport) {
	case "HTTP", "KAFKA":
		return &cfg, nil
	}
	return nil, fmt.Errorf(
		"unsupported UPDATES_TRANSPORT %q: expected one of [HTTP, KAFKA]",
		cfg.UpdatesTransport,
	)
}
