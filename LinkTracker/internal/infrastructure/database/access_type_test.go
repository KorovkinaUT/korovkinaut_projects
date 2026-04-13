package database_test

import (
	"context"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database"
	goqurepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database/goqu"
	sqlrepo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database/sql"
)

func TestNewRepositories_AccessTypeSelection(t *testing.T) {
	ctx := context.Background()

	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			// arrange
			container, cfg := startPostgresContainer(t, ctx)
			cfg.AccessType = accessType

			pool := newPool(t, ctx, cfg)
			t.Cleanup(func() {
				pool.Close()
				_ = container.Terminate(ctx)
			})

			applyMigrations(t, cfg)

			// act
			chatRepo, subscriptionRepo, err := database.NewRepositories(cfg, pool)
			if err != nil {
				t.Fatalf("failed to initialize repositories: %v", err)
			}

			// assert
			switch accessType {
			case "SQL":
				if _, ok := chatRepo.(*sqlrepo.ChatRepository); !ok {
					t.Errorf("expected sql chat repository, got %T", chatRepo)
				}
				if _, ok := subscriptionRepo.(*sqlrepo.SubscriptionRepository); !ok {
					t.Errorf("expected sql subscription repository, got %T", subscriptionRepo)
				}
			case "GOQU":
				if _, ok := chatRepo.(*goqurepo.ChatRepository); !ok {
					t.Errorf("expected goqu chat repository, got %T", chatRepo)
				}
				if _, ok := subscriptionRepo.(*goqurepo.SubscriptionRepository); !ok {
					t.Errorf("expected goqu subscription repository, got %T", subscriptionRepo)
				}
			}
		})
	}
}
