package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type ValkeyConfig struct {
	Addresses []string      `envconfig:"VALKEY_ADDRESSES" default:"localhost:6379"`
	TTL       time.Duration `envconfig:"VALKEY_TTL" default:"24h"`
	Timeout   time.Duration `envconfig:"VALKEY_TIMEOUT" default:"5s"`

	ClientSideCachingEnabled bool          `envconfig:"VALKEY_CLIENT_SIDE_CACHING_ENABLED" default:"false"`
	ClientSideCacheTTL       time.Duration `envconfig:"VALKEY_CLIENT_SIDE_CACHE_TTL" default:"1m"`
}

func LoadValkeyConfig() (*ValkeyConfig, error) {
	var cfg ValkeyConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	if len(cfg.Addresses) == 0 {
		cfg.Addresses = []string{"localhost:6379"}
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.ClientSideCacheTTL <= 0 {
		cfg.ClientSideCacheTTL = time.Minute
	}

	return &cfg, nil
}
