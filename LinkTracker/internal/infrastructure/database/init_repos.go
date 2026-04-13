package database

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	goqurepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database/goqu"
	sqlrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database/sql"
)

func NewRepositories(cfg *config.DatabaseConfig, pool *pgxpool.Pool) (repository.ChatRepository, repository.SubscriptionRepository, error) {
	switch strings.ToUpper(cfg.AccessType) {
	case "SQL":
		return sqlrepo.NewChatRepository(pool),
			sqlrepo.NewSubscriptionRepository(pool), nil

	case "GOQU":
		return goqurepo.NewChatRepository(pool),
			goqurepo.NewSubscriptionRepository(pool), nil

	default:
		return nil, nil, fmt.Errorf("unknown database access type: %q", cfg.AccessType)
	}
}
