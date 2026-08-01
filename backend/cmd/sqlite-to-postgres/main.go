package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type tableSpec struct {
	Name         string
	Columns      []string
	JSONColumns  map[string]bool
	TimeColumns  map[string]bool
	UnixColumns  map[string]bool
	BoolColumns  map[string]bool
	NullableTime map[string]bool
	SetSequence  bool
}

type tableManifest struct {
	Table        string `json:"table"`
	Rows         int64  `json:"rows"`
	SHA256       string `json:"sha256"`
	TargetRows   *int64 `json:"targetRows,omitempty"`
	TargetSHA256 string `json:"targetSha256,omitempty"`
	Verified     bool   `json:"verified,omitempty"`
	StartedAt    string `json:"startedAt"`
	FinishedAt   string `json:"finishedAt"`
}

type migrationManifest struct {
	StartedAt        string          `json:"startedAt"`
	FinishedAt       string          `json:"finishedAt"`
	ForeignKeysValid *bool           `json:"foreignKeysValid,omitempty"`
	Tables           []tableManifest `json:"tables"`
}

var orderedTables = []tableSpec{
	spec("users", true, []string{"id", "email", "password_hash", "nickname", "role", "province", "grade", "is_shadow", "banned_at", "banned_reason", "deleted_at", "created_at", "updated_at"}, nil, []string{"banned_at", "deleted_at", "created_at", "updated_at"}, nil, []string{"is_shadow"}),
	spec("auth_sessions", true, []string{"id", "user_id", "token_hash", "created_at", "expires_at", "revoked_at"}, nil, []string{"created_at", "expires_at", "revoked_at"}, nil, nil),
	spec("posts", true, []string{"id", "user_id", "author_name", "author_role", "title", "content", "image_urls", "tags", "track", "electives", "category", "grade", "province", "likes_count", "comments_count", "favorites_count", "created_at", "updated_at", "deleted_at"}, []string{"image_urls", "tags", "electives"}, []string{"created_at", "updated_at", "deleted_at"}, nil, nil),
	spec("comments", true, []string{"id", "post_id", "user_id", "author", "role", "content", "created_at", "deleted_at"}, nil, []string{"created_at", "deleted_at"}, nil, nil),
	spec("content_reports", true, []string{"id", "reporter_user_id", "target_type", "target_id", "reason", "detail", "status", "resolution_note", "resolved_at", "created_at", "updated_at"}, nil, []string{"resolved_at", "created_at", "updated_at"}, nil, nil),
	spec("subject_insights", true, []string{"id", "combination", "trend", "heat", "match_rate", "advice", "details", "metric_type", "unit", "province", "data_year", "source_name", "source_url", "scope", "sample_size", "captured_at", "methodology", "updated_at"}, nil, []string{"captured_at", "updated_at"}, nil, nil),
	spec("post_likes", false, []string{"user_id", "post_id", "created_at"}, nil, []string{"created_at"}, nil, nil),
	spec("post_favorites", false, []string{"user_id", "post_id", "created_at"}, nil, []string{"created_at"}, nil, nil),
	spec("follows", false, []string{"follower_id", "author_name", "created_at"}, nil, []string{"created_at"}, nil, nil),
	spec("user_profiles", false, []string{"user_id", "bio", "choice_profile", "created_at", "updated_at"}, []string{"choice_profile"}, []string{"created_at", "updated_at"}, nil, nil),
	spec("notifications", true, []string{"id", "recipient_user_id", "actor_user_id", "type", "title", "summary", "target_url", "created_at", "read_at"}, nil, []string{"created_at", "read_at"}, nil, nil),
	spec("direct_messages", true, []string{"id", "sender_user_id", "recipient_user_id", "content", "created_at", "read_at"}, nil, []string{"created_at", "read_at"}, nil, nil),
	spec("topics", true, []string{"id", "slug", "topic_tag", "title", "summary", "views_count", "posts_count", "created_at"}, nil, []string{"created_at"}, nil, nil),
	spec("topic_posts", false, []string{"topic_id", "post_id"}, nil, nil, nil, nil),
	spec("email_verification_codes", true, []string{"id", "email", "code_hash", "expires_at", "used_at", "failed_attempts", "created_at"}, nil, []string{"expires_at", "used_at", "created_at"}, nil, nil),
	spec("email_verification_attempts", true, []string{"id", "email", "client_ip", "created_at"}, nil, nil, []string{"created_at"}, nil),
	spec("upload_assets", false, []string{"id", "user_id", "asset_key", "file_name", "content_type", "ext", "size_bytes", "width", "height", "status", "created_at", "expires_at", "completed_at"}, nil, []string{"created_at", "expires_at", "completed_at"}, nil, nil),
	spec("admin_content_records", false, []string{"id", "module", "title", "content_type", "status", "scope", "owner", "tags", "summary", "url", "priority", "sort_order", "payload", "created_at", "updated_at", "deleted_at"}, []string{"tags", "payload"}, []string{"created_at", "updated_at", "deleted_at"}, nil, nil),
	spec("admin_audit_logs", true, []string{"id", "action", "record_id", "module", "detail", "actor", "created_at"}, nil, []string{"created_at"}, nil, nil),
	spec("content_sources", true, []string{"id", "post_id", "source_platform", "source_url", "source_note_id", "source_title", "source_author", "source_avatar_url", "source_likes", "source_comments", "source_favorites", "source_format", "transformation_note", "captured_at"}, nil, []string{"captured_at"}, nil, nil),
}

func main() {
	var sqlitePath string
	var postgresURL string
	var manifestPath string
	var dryRun bool
	flag.StringVar(&sqlitePath, "sqlite", "", "source SQLite database path")
	flag.StringVar(&postgresURL, "postgres", os.Getenv("DATABASE_URL"), "target PostgreSQL connection string")
	flag.StringVar(&manifestPath, "manifest", "", "optional path to write JSON row-count and SHA-256 manifest")
	flag.BoolVar(&dryRun, "dry-run", false, "read SQLite and produce manifest without writing PostgreSQL")
	flag.Parse()

	if sqlitePath == "" {
		exitf("-sqlite is required")
	}
	if !dryRun && postgresURL == "" {
		exitf("-postgres or DATABASE_URL is required unless -dry-run is set")
	}

	ctx := context.Background()
	manifest, err := run(ctx, sqlitePath, postgresURL, dryRun)
	if err != nil {
		exitf("%v", err)
	}
	if err := writeManifest(os.Stdout, manifest); err != nil {
		exitf("write manifest: %v", err)
	}
	if manifestPath != "" {
		file, err := os.Create(manifestPath)
		if err != nil {
			exitf("create manifest: %v", err)
		}
		if err := writeManifest(file, manifest); err != nil {
			_ = file.Close()
			exitf("write manifest file: %v", err)
		}
		if err := file.Close(); err != nil {
			exitf("close manifest file: %v", err)
		}
	}
}

func run(ctx context.Context, sqlitePath string, postgresURL string, dryRun bool) (migrationManifest, error) {
	startedAt := time.Now().UTC()
	source, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return migrationManifest{}, err
	}
	defer source.Close()
	if _, err := source.ExecContext(ctx, "PRAGMA query_only = ON"); err != nil {
		return migrationManifest{}, err
	}

	var target *sql.DB
	var tx *sql.Tx
	if !dryRun {
		target, err = sql.Open("pgx", postgresURL)
		if err != nil {
			return migrationManifest{}, err
		}
		defer target.Close()
		tx, err = target.BeginTx(ctx, nil)
		if err != nil {
			return migrationManifest{}, err
		}
		defer tx.Rollback()
	}

	manifest := migrationManifest{StartedAt: startedAt.Format(time.RFC3339Nano)}
	for _, table := range orderedTables {
		result, err := copyTable(ctx, source, tx, table, dryRun)
		if err != nil {
			return migrationManifest{}, fmt.Errorf("copy %s: %w", table.Name, err)
		}
		manifest.Tables = append(manifest.Tables, result)
	}
	if tx != nil {
		for index, table := range orderedTables {
			targetManifest, err := readTargetManifest(ctx, tx, table)
			if err != nil {
				return migrationManifest{}, fmt.Errorf("verify %s: %w", table.Name, err)
			}
			sourceManifest := &manifest.Tables[index]
			sourceManifest.TargetRows = &targetManifest.Rows
			sourceManifest.TargetSHA256 = targetManifest.SHA256
			sourceManifest.Verified = sourceManifest.Rows == targetManifest.Rows && sourceManifest.SHA256 == targetManifest.SHA256
			if !sourceManifest.Verified {
				return migrationManifest{}, fmt.Errorf("verify %s: source rows/hash %d/%s differ from target %d/%s", table.Name, sourceManifest.Rows, sourceManifest.SHA256, targetManifest.Rows, targetManifest.SHA256)
			}
		}
		foreignKeysValid, err := verifyForeignKeys(ctx, tx)
		if err != nil {
			return migrationManifest{}, fmt.Errorf("verify foreign keys: %w", err)
		}
		manifest.ForeignKeysValid = &foreignKeysValid
		if !foreignKeysValid {
			return migrationManifest{}, fmt.Errorf("verify foreign keys: PostgreSQL contains unvalidated foreign-key constraints")
		}
		if err := tx.Commit(); err != nil {
			return migrationManifest{}, err
		}
	}
	manifest.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return manifest, nil
}

type rowQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readTargetManifest(ctx context.Context, queryer rowQueryer, table tableSpec) (tableManifest, error) {
	rows, err := queryer.QueryContext(ctx, selectSQL(table))
	if err != nil {
		return tableManifest{}, err
	}
	defer rows.Close()

	hasher := sha256.New()
	var count int64
	for rows.Next() {
		raw := make([]any, len(table.Columns))
		dest := make([]any, len(table.Columns))
		for i := range raw {
			dest[i] = &raw[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return tableManifest{}, err
		}
		values, err := normalizeTargetRow(table, raw)
		if err != nil {
			return tableManifest{}, err
		}
		hashRow(hasher, table.Columns, values)
		count++
	}
	if err := rows.Err(); err != nil {
		return tableManifest{}, err
	}
	return tableManifest{Table: table.Name, Rows: count, SHA256: hex.EncodeToString(hasher.Sum(nil))}, nil
}

func normalizeTargetRow(table tableSpec, raw []any) ([]any, error) {
	values := make([]any, len(raw))
	for index, value := range raw {
		if value == nil {
			continue
		}
		column := table.Columns[index]
		if table.BoolColumns[column] {
			normalized, err := sqliteBool(value)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", table.Name, column, err)
			}
			values[index] = normalized
			continue
		}
		if table.JSONColumns[column] {
			normalized, err := canonicalJSON(fmt.Sprint(normalizeSQLiteValue(value)))
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", table.Name, column, err)
			}
			values[index] = normalized
			continue
		}
		if table.Name == "subject_insights" && column == "match_rate" {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(normalizeSQLiteValue(value))), 64)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", table.Name, column, err)
			}
			values[index] = parsed
			continue
		}
		values[index] = normalizeSQLiteValue(value)
	}
	return values, nil
}

func verifyForeignKeys(ctx context.Context, queryer rowQueryer) (bool, error) {
	var unvalidated int
	err := queryer.QueryRowContext(ctx, `SELECT COUNT(*) FROM pg_constraint WHERE contype = 'f' AND NOT convalidated`).Scan(&unvalidated)
	return unvalidated == 0, err
}

func copyTable(ctx context.Context, source *sql.DB, tx *sql.Tx, table tableSpec, dryRun bool) (tableManifest, error) {
	startedAt := time.Now().UTC()
	if err := requireSQLiteSchema(ctx, source, table); err != nil {
		return tableManifest{}, err
	}
	rows, err := source.QueryContext(ctx, selectSQL(table))
	if err != nil {
		return tableManifest{}, err
	}
	defer rows.Close()

	hasher := sha256.New()
	insertSQL := insertSQL(table)
	var count int64
	for rows.Next() {
		raw := make([]any, len(table.Columns))
		dest := make([]any, len(table.Columns))
		for i := range raw {
			dest[i] = &raw[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return tableManifest{}, err
		}
		values, err := convertRow(table, raw)
		if err != nil {
			return tableManifest{}, err
		}
		hashRow(hasher, table.Columns, values)
		if !dryRun {
			if _, err := tx.ExecContext(ctx, insertSQL, values...); err != nil {
				return tableManifest{}, err
			}
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return tableManifest{}, err
	}
	if !dryRun && table.SetSequence && count > 0 {
		if err := setSequence(ctx, tx, table.Name); err != nil {
			return tableManifest{}, err
		}
	}
	return tableManifest{
		Table:      table.Name,
		Rows:       count,
		SHA256:     hex.EncodeToString(hasher.Sum(nil)),
		StartedAt:  startedAt.Format(time.RFC3339Nano),
		FinishedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

func spec(name string, setSequence bool, columns []string, jsonColumns []string, timeColumns []string, unixColumns []string, boolColumns []string) tableSpec {
	return tableSpec{
		Name:         name,
		Columns:      columns,
		JSONColumns:  toSet(jsonColumns),
		TimeColumns:  toSet(timeColumns),
		UnixColumns:  toSet(unixColumns),
		BoolColumns:  toSet(boolColumns),
		NullableTime: toSet(timeColumns),
		SetSequence:  setSequence,
	}
}

func requireSQLiteSchema(ctx context.Context, db *sql.DB, table tableSpec) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table.Name).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("source table %q does not exist; run the current SQLite app once before migration", table.Name)
	}
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+quoteSQLiteIdent(table.Name)+")")
	if err != nil {
		return err
	}
	defer rows.Close()
	columns := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, column := range table.Columns {
		if !columns[column] {
			return fmt.Errorf("source table %q is missing column %q; start the current application once to apply the SQLite compatibility migration before exporting", table.Name, column)
		}
	}
	return nil
}

func selectSQL(table tableSpec) string {
	columns := quoteColumns(table.Columns)
	return fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", strings.Join(columns, ", "), quoteIdent(table.Name), orderBy(table))
}

func insertSQL(table tableSpec) string {
	placeholders := make([]string, len(table.Columns))
	for i, column := range table.Columns {
		placeholder := fmt.Sprintf("$%d", i+1)
		if table.JSONColumns[column] {
			placeholder += "::jsonb"
		}
		placeholders[i] = placeholder
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteIdent(table.Name), strings.Join(quoteColumns(table.Columns), ", "), strings.Join(placeholders, ", "))
}

func orderBy(table tableSpec) string {
	for _, column := range []string{"id", "user_id", "topic_id"} {
		if contains(table.Columns, column) {
			return quoteIdent(column)
		}
	}
	return strings.Join(quoteColumns(table.Columns), ", ")
}

func setSequence(ctx context.Context, tx *sql.Tx, table string) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf("SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1), true)", table, quoteIdent(table)))
	return err
}

func convertRow(table tableSpec, raw []any) ([]any, error) {
	values := make([]any, len(raw))
	for i, value := range raw {
		column := table.Columns[i]
		if value == nil {
			values[i] = nil
			continue
		}
		normalized := normalizeSQLiteValue(value)
		if table.JSONColumns[column] {
			text := fmt.Sprint(normalized)
			if text == "" {
				text = jsonDefault(column)
			}
			canonical, err := canonicalJSON(text)
			if err != nil {
				return nil, fmt.Errorf("%s.%s has invalid JSON: %w", table.Name, column, err)
			}
			values[i] = canonical
			continue
		}
		if table.UnixColumns[column] {
			parsed, err := unixTime(normalized)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", table.Name, column, err)
			}
			values[i] = parsed
			continue
		}
		if table.BoolColumns[column] {
			parsed, err := sqliteBool(normalized)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", table.Name, column, err)
			}
			values[i] = parsed
			continue
		}
		if table.TimeColumns[column] {
			parsed, err := parseSQLiteTime(normalized)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", table.Name, column, err)
			}
			values[i] = parsed
			continue
		}
		values[i] = normalized
	}
	return values, nil
}

func canonicalJSON(value string) (string, error) {
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func sqliteBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case int64:
		return typed != 0, nil
	case int:
		return typed != 0, nil
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "1" || strings.EqualFold(text, "true") {
			return true, nil
		}
		if text == "0" || strings.EqualFold(text, "false") || text == "" {
			return false, nil
		}
		return false, fmt.Errorf("unsupported bool %q", text)
	}
}

func parseSQLiteTime(value any) (any, error) {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.UTC().Round(time.Microsecond), nil
		}
	}
	return nil, fmt.Errorf("unsupported time %q", text)
}

func unixTime(value any) (time.Time, error) {
	switch typed := value.(type) {
	case int64:
		return time.Unix(typed, 0).UTC(), nil
	case int:
		return time.Unix(int64(typed), 0).UTC(), nil
	case float64:
		return time.Unix(int64(typed), 0).UTC(), nil
	default:
		seconds, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(value)), 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(seconds, 0).UTC(), nil
	}
}

func normalizeSQLiteValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case bool:
		if typed {
			return int64(1)
		}
		return int64(0)
	default:
		return typed
	}
}

func hashRow(hasher io.Writer, columns []string, values []any) {
	for i, column := range columns {
		_, _ = fmt.Fprintf(hasher, "%s=", column)
		_, _ = fmt.Fprint(hasher, canonicalValue(values[i]))
		if i < len(columns)-1 {
			_, _ = io.WriteString(hasher, "\x1f")
		}
	}
	_, _ = io.WriteString(hasher, "\n")
}

func canonicalValue(value any) string {
	if value == nil {
		return "<null>"
	}
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(typed)
	}
}

func writeManifest(writer io.Writer, manifest migrationManifest) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(manifest)
}

func jsonDefault(column string) string {
	if column == "payload" || column == "choice_profile" {
		return "{}"
	}
	return "[]"
}

func toSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func quoteColumns(columns []string) []string {
	quoted := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = quoteIdent(column)
	}
	return quoted
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteSQLiteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func exitf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
