package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type DatabaseConfig struct {
	Host           string        `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port           int           `envconfig:"POSTGRES_PORT" default:"5432"`
	User           string        `envconfig:"POSTGRES_USER" default:"postgres"`
	Password       string        `envconfig:"POSTGRES_PASSWORD" default:"postgres"`
	DBName         string        `envconfig:"POSTGRES_DB" default:"link_tracker"`
	SSLMode        string        `envconfig:"POSTGRES_SSLMODE" default:"disable"`
	AccessType     string        `envconfig:"DB_ACCESS_TYPE" default:"SQL"`
	ConnectTimeout time.Duration `envconfig:"DB_CONNECT_TIMEOUT" default:"5s"`
}

func (c *DatabaseConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}

func LoadDatabaseConfig() (*DatabaseConfig, error) {
	var cfg DatabaseConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	switch strings.ToUpper(cfg.AccessType) {
	case "SQL", "GOQU":
		return &cfg, nil
	}
	return nil, fmt.Errorf(
		"unsupported DATABASE_ACCESS_TYPE %q: expected one of [SQL, GOQU]",
		cfg.AccessType,
	)
}
