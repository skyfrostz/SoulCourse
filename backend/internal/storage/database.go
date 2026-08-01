package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"subject-choice-forum/backend/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const RequiredPostgresSchemaVersion int64 = 1

type Database struct {
	DB     *sql.DB
	Driver string
}

func NewDatabase(ctx context.Context, cfg config.Config) (*Database, error) {
	switch cfg.DatabaseDriver {
	case "sqlite":
		db, err := NewSQLiteDB(cfg)
		if err != nil {
			return nil, err
		}
		return &Database{DB: db, Driver: "sqlite"}, nil
	case "postgres":
		db, err := newPostgresDB(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return &Database{DB: db, Driver: "postgres"}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.DatabaseDriver)
	}
}

func newPostgresDB(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	pgxConfig, err := pgx.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL DATABASE_URL: %w", err)
	}
	if pgxConfig.RuntimeParams == nil {
		pgxConfig.RuntimeParams = make(map[string]string)
	}
	pgxConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(cfg.DatabaseQueryTimeout.Milliseconds(), 10)

	db := sql.OpenDB(stdlib.GetConnector(*pgxConfig))
	db.SetMaxOpenConns(cfg.DatabaseMaxOpenConns)
	db.SetMaxIdleConns(cfg.DatabaseMaxIdleConns)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	connectCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
	defer cancel()
	if err := db.PingContext(connectCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect PostgreSQL: %w", err)
	}
	if err := VerifyPostgresSchema(connectCtx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func VerifyPostgresSchema(ctx context.Context, db *sql.DB) error {
	var version int64
	var dirty bool
	err := db.QueryRowContext(ctx, `
		SELECT version_id, is_applied = false
		FROM goose_db_version
		ORDER BY id DESC
		LIMIT 1`).Scan(&version, &dirty)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL goose schema: %w", err)
	}
	if dirty || version != RequiredPostgresSchemaVersion {
		return fmt.Errorf("PostgreSQL schema is not ready: version=%d dirty=%t required=%d", version, dirty, RequiredPostgresSchemaVersion)
	}

	var missingList string
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(string_agg(required.name, ',' ORDER BY required.name), '')
		FROM (VALUES
			('users'), ('posts'), ('auth_sessions'), ('provinces'),
			('sources'), ('policies'), ('requirements'), ('upload_assets')
		) AS required(name)
		WHERE to_regclass('public.' || required.name) IS NULL`).Scan(&missingList); err != nil {
		return fmt.Errorf("verify PostgreSQL required tables: %w", err)
	}
	if missingList != "" {
		return fmt.Errorf("PostgreSQL schema is missing required tables: %s", missingList)
	}
	return nil
}
