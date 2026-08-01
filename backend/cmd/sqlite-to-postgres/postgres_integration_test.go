package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/storage"
)

func TestPostgresMigrationIntegration(t *testing.T) {
	postgresURL := os.Getenv("POSTGRES_INTEGRATION_URL")
	if postgresURL == "" {
		t.Skip("POSTGRES_INTEGRATION_URL is not set")
	}

	tempDir := t.TempDir()
	sqlitePath := filepath.Join(tempDir, "source.db")
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     sqlitePath,
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
		JWTSecret:      "integration-test-secret",
	})
	if err != nil {
		t.Fatalf("create SQLite fixture: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close SQLite fixture: %v", err)
	}

	manifest, err := run(context.Background(), sqlitePath, postgresURL, false)
	if err != nil {
		t.Fatalf("run migration: %v", err)
	}
	if manifest.ForeignKeysValid == nil || !*manifest.ForeignKeysValid {
		t.Fatalf("foreign key verification missing or false: %#v", manifest.ForeignKeysValid)
	}
	if len(manifest.Tables) != len(orderedTables) {
		t.Fatalf("verified tables = %d, want %d", len(manifest.Tables), len(orderedTables))
	}
	for _, table := range manifest.Tables {
		if !table.Verified || table.TargetRows == nil || *table.TargetRows != table.Rows || table.TargetSHA256 != table.SHA256 {
			t.Fatalf("table %s was not verified: %#v", table.Table, table)
		}
	}
}
