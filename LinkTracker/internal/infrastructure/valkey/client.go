package valkey

import (
	"github.com/valkey-io/valkey-go"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func NewClient(cfg *config.ValkeyConfig) (valkey.Client, error) {
	return valkey.NewClient(valkey.ClientOption{
		InitAddress:  cfg.Addresses,
		DisableCache: !cfg.ClientSideCachingEnabled,
	})
}
