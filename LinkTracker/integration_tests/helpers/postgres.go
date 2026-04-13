package helpers

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database"
)

type TestDatabase struct {
	Config    *config.DatabaseConfig
	Container testcontainers.Container
}

func NewTestDatabase(t *testing.T, accessType string) *TestDatabase {
	t.Helper()

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "postgres:16",
			Env: map[string]string{
				"POSTGRES_DB":       "link_tracker_test",
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
			},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForLog("database system is ready to accept connections"),
			).WithDeadline(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("get postgres host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("get postgres mapped port: %v", err)
	}

	cfg := &config.DatabaseConfig{
		Host:           host,
		Port:           port.Int(),
		User:           "postgres",
		Password:       "postgres",
		DBName:         "link_tracker_test",
		SSLMode:        "disable",
		AccessType:     accessType,
		ConnectTimeout: 5 * time.Second,
	}

	return &TestDatabase{
		Config:    cfg,
		Container: container,
	}
}

func (db *TestDatabase) NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), db.Config.ConnectTimeout)
	defer cancel()

	pool, err := database.NewPool(ctx, db.Config)

	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}

	return pool
}

func (db *TestDatabase) Close(t *testing.T) {
	t.Helper()

	if err := db.Container.Terminate(context.Background()); err != nil {
		t.Fatalf("terminate postgres container: %v", err)
	}
}
