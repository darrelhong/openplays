// Package testdb provides a reusable in-memory SQLite database for integration tests.
// It applies all goose migrations from db/migrations/ so tests run against the real schema.
package testdb

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/glebarez/sqlite"
	"github.com/pressly/goose/v3"
)

// New creates a fresh in-memory SQLite database with all migrations applied.
// The database is automatically closed when the test completes.
func New(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("testdb: open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	migrationsDir := findMigrationsDir(t)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("testdb: set dialect: %v", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		t.Fatalf("testdb: goose up: %v", err)
	}

	return db
}

// findMigrationsDir locates db/migrations/ relative to the project root.
func findMigrationsDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("testdb: cannot determine source file path")
	}

	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "db", "migrations")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("testdb: cannot find project root (no go.mod found)")
		}
		dir = parent
	}
}
