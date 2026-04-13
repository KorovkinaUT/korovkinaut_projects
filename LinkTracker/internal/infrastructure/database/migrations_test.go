package database_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database"
)

func TestMigrations_ApplyOnCleanDatabase(t *testing.T) {
	ctx := context.Background()

	// arrange
	container, cfg := startPostgresContainer(t, ctx)
	pool := newPool(t, ctx, cfg)
	t.Cleanup(func() {
		pool.Close()
		_ = container.Terminate(ctx)
	})

	existsBefore, err := tableExists(ctx, pool, "chats")
	if err != nil {
		t.Fatalf("check table before migrations: %v", err)
	}
	if existsBefore {
		t.Fatal("expected clean database before migrations")
	}

	// act
	applyMigrations(t, cfg)

	// assert
	for _, tableName := range []string{"chats", "links", "link_chat", "link_tag"} {
		exists, err := tableExists(ctx, pool, tableName)
		if err != nil {
			t.Fatalf("check table %q after migrations: %v", tableName, err)
		}
		if !exists {
			t.Fatalf("expected table %q to exist after migrations", tableName)
		}
	}

	cfg.AccessType = "SQL"
	chatRepo, _, err := database.NewRepositories(cfg, pool)
	if err != nil {
		t.Fatalf("failed to initialize repositories: %v", err)
	}

	if err := chatRepo.Register(ctx, 1); err != nil {
		t.Fatalf("check repository usability after migrations: %v", err)
	}
}

func startPostgresContainer(t *testing.T, ctx context.Context) (testcontainers.Container, *config.DatabaseConfig) {
	t.Helper()

	const (
		dbName   = "link_tracker_test"
		user     = "postgres"
		password = "postgres"
	)

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       dbName,
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": password,
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
	}

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("get postgres container host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("get postgres mapped port: %v", err)
	}

	port, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("parse postgres mapped port: %v", err)
	}

	cfg := &config.DatabaseConfig{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   password,
		DBName:     dbName,
		SSLMode:    "disable",
		AccessType: "SQL",
	}

	return container, cfg
}

func tableExists(ctx context.Context, pool *pgxpool.Pool, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`

	var exists bool
	if err := pool.QueryRow(ctx, query, tableName).Scan(&exists); err != nil {
		return false, fmt.Errorf("check table exists: %w", err)
	}

	return exists, nil
}
