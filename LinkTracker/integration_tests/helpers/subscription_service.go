package helpers

import (
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database"
)

func NewTestBaseSubscriptionService(t *testing.T, db *TestDatabase) service.SubscriptionService {
	t.Helper()

	return NewTestSubscriptionService(t, db, false, nil)
}

func NewTestCachedSubscriptionService(
	t *testing.T,
	db *TestDatabase,
	listCache service.ListCache,
) service.SubscriptionService {
	t.Helper()

	return NewTestSubscriptionService(t, db, true, listCache)
}

func NewTestSubscriptionService(
	t *testing.T,
	db *TestDatabase,
	cacheEnabled bool,
	listCache service.ListCache,
) service.SubscriptionService {
	t.Helper()

	pool := db.NewPool(t)

	chatRepository, subscriptionRepository, err := database.NewRepositories(db.Config, pool)
	if err != nil {
		t.Fatalf("failed to initialize repositories: %v", err)
	}

	return service.NewSubscriptionService(
		cacheEnabled,
		chatRepository,
		subscriptionRepository,
		listCache,
		nil,
	)
}
