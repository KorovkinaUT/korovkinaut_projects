package database_test

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database"
)

type testEnv struct {
	cfg              *config.DatabaseConfig
	pool             *pgxpool.Pool
	chatRepo         repository.ChatRepository
	subscriptionRepo repository.SubscriptionRepository
}

func TestChatRepository_RegisterAndExists(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(101)

			// act
			err := env.chatRepo.Register(ctx, chatID)
			if err != nil {
				t.Fatalf("register chat: %v", err)
			}

			exists, err := env.chatRepo.Exists(ctx, chatID)
			if err != nil {
				t.Fatalf("check chat exists: %v", err)
			}

			// assert
			if !exists {
				t.Error("expected registered chat to exist")
			}
		})
	}
}

func TestChatRepository_RegisterDuplicateReturnsError(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(102)

			if err := env.chatRepo.Register(ctx, chatID); err != nil {
				t.Fatalf("prepare chat registration: %v", err)
			}

			// act
			err := env.chatRepo.Register(ctx, chatID)

			// assert
			if !errors.Is(err, repository.ErrChatAlreadyExists) {
				t.Errorf("expected ErrChatAlreadyExists, got %v", err)
			}
		})
	}
}

func TestChatRepository_DeleteRemovesChat(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(103)

			if err := env.chatRepo.Register(ctx, chatID); err != nil {
				t.Fatalf("prepare chat registration: %v", err)
			}

			// act
			err := env.chatRepo.Delete(ctx, chatID)
			if err != nil {
				t.Fatalf("delete chat: %v", err)
			}

			exists, err := env.chatRepo.Exists(ctx, chatID)
			if err != nil {
				t.Fatalf("check chat exists after delete: %v", err)
			}

			// assert
			if exists {
				t.Error("expected deleted chat to be absent")
			}
		})
	}
}

func TestChatRepository_DeleteMissingChatReturnsError(t *testing.T) {
	for _, accessType := range []string{"SQL", "GOQU"} {
		t.Run(accessType, func(t *testing.T) {
			env := newTestEnv(t, accessType)
			ctx := context.Background()

			// arrange
			chatID := int64(104)

			// act
			err := env.chatRepo.Delete(ctx, chatID)

			// assert
			if !errors.Is(err, repository.ErrChatNotFound) {
				t.Errorf("expected ErrChatNotFound, got %v", err)
			}
		})
	}
}

func newTestEnv(t *testing.T, accessType string) *testEnv {
	t.Helper()

	ctx := context.Background()

	container, cfg := startPostgresContainer(t, ctx)
	cfg.AccessType = accessType

	pool := newPool(t, ctx, cfg)
	applyMigrations(t, cfg)

	chatRepo, subscriptionRepo, err := database.NewRepositories(cfg, pool)
	if err != nil {
		t.Fatalf("failed to initialize repositories: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_ = container.Terminate(ctx)
	})

	return &testEnv{
		cfg:              cfg,
		pool:             pool,
		chatRepo:         chatRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

func newPool(t *testing.T, ctx context.Context, cfg *config.DatabaseConfig) *pgxpool.Pool {
	t.Helper()

	pool, err := database.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}

	return pool
}

func applyMigrations(t *testing.T, cfg *config.DatabaseConfig) {
	t.Helper()

	migrationsPath := migrationsDir(t)

	m, err := migrate.New(
		"file://"+migrationsPath,
		cfg.URL(),
	)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			t.Fatalf("close migration source: %v", sourceErr)
		}
		if databaseErr != nil {
			t.Fatalf("close migration database: %v", databaseErr)
		}
	}()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("apply migrations: %v", err)
	}
}

func migrationsDir(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../migrations"))
}
