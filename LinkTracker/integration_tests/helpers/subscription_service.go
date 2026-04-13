package helpers

import (
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database"
)

func NewTestSubscriptionService(t *testing.T, db *TestDatabase) *service.SubscriptionService {
	t.Helper()

	pool := db.NewPool(t)

	chatRepository, subscriptionRepository, err := database.NewRepositories(db.Config, pool)
	if err != nil {
		t.Fatalf("failed to initialize repositories: %v", err)
	}

	return service.NewSubscriptionService(
		chatRepository,
		subscriptionRepository,
	)
}