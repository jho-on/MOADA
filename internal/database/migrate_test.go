package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestMigrateUp requires a disposable PostgreSQL database and is skipped
// unless MOADA_TEST_DATABASE_URL is set.
func TestMigrateUp(t *testing.T) {
	databaseURL := os.Getenv("MOADA_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set MOADA_TEST_DATABASE_URL to run the migration integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := Migrate(ctx, databaseURL, "up"); err != nil {
		t.Fatalf("Migrate(up) error = %v", err)
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	var installed bool
	if err := conn.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'citext')",
	).Scan(&installed); err != nil {
		t.Fatalf("query extension: %v", err)
	}
	if !installed {
		t.Error("citext extension was not installed by the migration")
	}
}
