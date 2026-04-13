package helpers

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func ApplyMigrations(t *testing.T, db *TestDatabase) {
	t.Helper()

	migrationsPath := migrationsDir(t)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		db.Config.URL(),
	)
	if err != nil {
		t.Fatalf("create migrate instance: %v", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			t.Errorf("close migrate source: %v", sourceErr)
		}
		if databaseErr != nil {
			t.Errorf("close migrate database: %v", databaseErr)
		}
	}()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatalf("apply migrations: %v", err)
	}
}

func migrationsDir(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("get current file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
}
