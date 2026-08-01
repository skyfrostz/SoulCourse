package repository

import (
	"database/sql"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/storage"

	_ "modernc.org/sqlite"
)

func TestNewForumRepositoryRejectsInvalidDatabase(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		name     string
		database *storage.Database
		want     string
	}{
		{name: "nil database", want: "database is required"},
		{name: "nil sql handle", database: &storage.Database{Driver: "sqlite"}, want: "database is required"},
		{name: "empty driver", database: &storage.Database{DB: db}, want: "unsupported database driver"},
		{name: "unknown driver", database: &storage.Database{DB: db, Driver: "mysql"}, want: "unsupported database driver: mysql"},
		{name: "driver is case sensitive", database: &storage.Database{DB: db, Driver: "Postgres"}, want: "unsupported database driver: Postgres"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, err := NewForumRepository(test.database)
			if repository != nil {
				t.Fatalf("repository = %T, want nil", repository)
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestNewForumRepositorySupportsSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repository, err := NewForumRepository(&storage.Database{DB: db, Driver: "sqlite"})
	if err != nil {
		t.Fatalf("NewForumRepository returned error: %v", err)
	}
	if repository == nil {
		t.Fatal("expected SQLite repository")
	}
}

func TestNewForumRepositorySupportsPostgres(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repository, err := NewForumRepository(&storage.Database{DB: db, Driver: "postgres"})
	if err != nil {
		t.Fatalf("NewForumRepository returned error: %v", err)
	}
	if repository == nil {
		t.Fatal("expected PostgreSQL repository")
	}
}

func TestNewUploadCleanupRepositorySupportsConfiguredDrivers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, driver := range []string{"sqlite", "postgres"} {
		t.Run(driver, func(t *testing.T) {
			repository, err := NewUploadCleanupRepository(&storage.Database{DB: db, Driver: driver})
			if err != nil || repository == nil {
				t.Fatalf("NewUploadCleanupRepository(%q) = %T, %v", driver, repository, err)
			}
		})
	}
}

func TestNewUploadCleanupRepositoryRejectsInvalidDatabase(t *testing.T) {
	if repository, err := NewUploadCleanupRepository(nil); repository != nil || err == nil || !strings.Contains(err.Error(), "database is required") {
		t.Fatalf("nil database = %T, %v", repository, err)
	}
}
