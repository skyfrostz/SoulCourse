package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
)

type schemaDriver struct {
	version      int64
	dirty        bool
	missing      string
	versionError error
	tablesError  error
}

type schemaConn struct{ driver *schemaDriver }
type schemaRows struct {
	columns []string
	values  [][]driver.Value
}

var schemaDriverID atomic.Uint64

func (d *schemaDriver) Open(string) (driver.Conn, error) { return &schemaConn{driver: d}, nil }
func (c *schemaConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (c *schemaConn) Close() error              { return nil }
func (c *schemaConn) Begin() (driver.Tx, error) { return nil, errors.New("transaction not supported") }
func (c *schemaConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "goose_db_version") {
		if c.driver.versionError != nil {
			return nil, c.driver.versionError
		}
		return &schemaRows{columns: []string{"version_id", "dirty"}, values: [][]driver.Value{{c.driver.version, c.driver.dirty}}}, nil
	}
	if strings.Contains(query, "to_regclass") {
		if c.driver.tablesError != nil {
			return nil, c.driver.tablesError
		}
		return &schemaRows{columns: []string{"missing"}, values: [][]driver.Value{{c.driver.missing}}}, nil
	}
	return nil, fmt.Errorf("unexpected query: %s", query)
}
func (r *schemaRows) Columns() []string { return r.columns }
func (r *schemaRows) Close() error      { return nil }
func (r *schemaRows) Next(dest []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(dest, r.values[0])
	r.values = r.values[1:]
	return nil
}

func openSchemaDB(t *testing.T, fake *schemaDriver) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("schema-test-%d", schemaDriverID.Add(1))
	sql.Register(name, fake)
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestNewDatabaseRejectsUnsupportedDriver(t *testing.T) {
	_, err := NewDatabase(context.Background(), config.Config{DatabaseDriver: "mysql"})
	if err == nil || !strings.Contains(err.Error(), "unsupported database driver") {
		t.Fatalf("error = %v, want unsupported driver", err)
	}
}

func TestNewPostgresDBRejectsMalformedURLBeforeConnecting(t *testing.T) {
	_, err := newPostgresDB(context.Background(), config.Config{
		DatabaseURL:            "://bad-url",
		DatabaseMaxOpenConns:   20,
		DatabaseMaxIdleConns:   5,
		DatabaseConnectTimeout: time.Second,
		DatabaseQueryTimeout:   time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "parse PostgreSQL DATABASE_URL") {
		t.Fatalf("error = %v, want URL parse error", err)
	}
}

func TestVerifyPostgresSchemaBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		driver    schemaDriver
		wantError string
	}{
		{name: "ready", driver: schemaDriver{version: RequiredPostgresSchemaVersion}},
		{name: "dirty", driver: schemaDriver{version: RequiredPostgresSchemaVersion, dirty: true}, wantError: "dirty=true"},
		{name: "behind", driver: schemaDriver{version: RequiredPostgresSchemaVersion - 1}, wantError: "version=0"},
		{name: "ahead", driver: schemaDriver{version: RequiredPostgresSchemaVersion + 1}, wantError: "version=2"},
		{name: "missing tables", driver: schemaDriver{version: RequiredPostgresSchemaVersion, missing: "policies,users"}, wantError: "missing required tables: policies,users"},
		{name: "version query failure", driver: schemaDriver{versionError: errors.New("version unavailable")}, wantError: "verify PostgreSQL goose schema: version unavailable"},
		{name: "table query failure", driver: schemaDriver{version: RequiredPostgresSchemaVersion, tablesError: errors.New("catalog unavailable")}, wantError: "verify PostgreSQL required tables: catalog unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openSchemaDB(t, &test.driver)
			err := VerifyPostgresSchema(context.Background(), db)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("VerifyPostgresSchema() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("VerifyPostgresSchema() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestNewDatabaseUsesSQLiteConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	database, err := NewDatabase(context.Background(), config.Config{
		DatabaseDriver: "sqlite",
		SQLitePath:     filepath.Join(tempDir, "configured.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer database.DB.Close()
	if database.Driver != "sqlite" {
		t.Fatalf("driver = %q, want sqlite", database.Driver)
	}
	if err := database.DB.PingContext(context.Background()); err != nil {
		t.Fatalf("configured SQLite database is not usable: %v", err)
	}
}

func TestNewPostgresDBHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	_, err := newPostgresDB(ctx, config.Config{
		DatabaseURL:            "postgres://user:password@127.0.0.1:1/database?sslmode=disable",
		DatabaseMaxOpenConns:   3,
		DatabaseMaxIdleConns:   1,
		DatabaseConnectTimeout: time.Minute,
		DatabaseQueryTimeout:   250 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "connect PostgreSQL") {
		t.Fatalf("error = %v, want PostgreSQL connection error", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("canceled context took %s", elapsed)
	}
}

func TestNewDatabaseRejectsSQLiteSetupFailure(t *testing.T) {
	parent := t.TempDir()
	blocker := filepath.Join(parent, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := NewDatabase(context.Background(), config.Config{
		DatabaseDriver: "sqlite",
		SQLitePath:     filepath.Join(blocker, "app.db"),
		MediaUploadDir: filepath.Join(parent, "uploads"),
	})
	if err == nil {
		t.Fatal("NewDatabase unexpectedly succeeded with an unusable SQLite parent")
	}
}

func TestNewDatabasePropagatesPostgresConfigurationFailure(t *testing.T) {
	_, err := NewDatabase(context.Background(), config.Config{
		DatabaseDriver: "postgres",
		DatabaseURL:    "://bad-url",
	})
	if err == nil || !strings.Contains(err.Error(), "parse PostgreSQL DATABASE_URL") {
		t.Fatalf("error = %v, want PostgreSQL URL parse error", err)
	}
}
