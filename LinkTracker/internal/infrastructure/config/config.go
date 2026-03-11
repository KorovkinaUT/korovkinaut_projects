package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// BotConfig contains bot configuration
type BotConfig struct {
	AppTelegramToken   string        `envconfig:"APP_TELEGRAM_TOKEN" required:"true"`
	PollTimeoutSeconds int           `envconfig:"POLL_TIMEOUT_SECONDS" default:"30s"`
	BotHost            string        `envconfig:"BOT_HOST" default:"localhost"`
	BotPort            int           `envconfig:"BOT_PORT" default:"8080"`
	ScrapperHost       string        `envconfig:"SCRAPPER_HOST" default:"localhost"`
	ScrapperPort       int           `envconfig:"SCRAPPER_PORT" default:"8081"`
	HttpTimeout        time.Duration `envconfig:"HTTP_TIMEOUT" default:"5s"`
}

func (c *BotConfig) BotAddress() string {
	return fmt.Sprintf(":%d", c.BotPort)
}

func (c *BotConfig) ScrapperBaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.ScrapperHost, c.ScrapperPort)
}

// Load, reads and validates required configuration
func LoadBotConfig() (*BotConfig, error) {
	var cfg BotConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ScrapperConfig contains scrapper configuration
type ScrapperConfig struct {
	SchedulerInterval time.Duration `envconfig:"SCHEDULER_INTERVAL" default:"1m"`
	BotHost string `envconfig:"BOT_HOST" default:"localhost"`
	BotPort int    `envconfig:"BOT_PORT" default:"8080"`
	ScrapperHost string `envconfig:"SCRAPPER_HOST" default:"localhost"`
	ScrapperPort int    `envconfig:"SCRAPPER_PORT" default:"8081"`
	HttpTimeout time.Duration `envconfig:"HTTP_TIMEOUT" default:"5s"`
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
	return &cfg, nil
}
