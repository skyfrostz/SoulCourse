package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"subject-choice-forum/backend/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestAdminContentPostgresJourney(t *testing.T) {
	postgresURL := os.Getenv("POSTGRES_ADMIN_TEST_URL")
	if postgresURL == "" {
		t.Skip("POSTGRES_ADMIN_TEST_URL is not set")
	}
	db, err := sql.Open("pgx", postgresURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	handler := &AdminHandler{cfg: config.Config{DatabaseDriver: "postgres"}, db: db}
	ctx := context.Background()
	payload := json.RawMessage(`{"content":"后台 PostgreSQL 同步正文","track":"physics","electives":["chemistry","biology"],"category":"question","grade":"高一","province":"广东","imageUrls":[]}`)

	record, err := handler.upsertContent(ctx, AdminContentInput{
		ID: "postgres-admin-post", Module: "posts", Title: "后台 PostgreSQL 内容",
		Type: "提问", Status: "已上架", Scope: "广东", Owner: "PG后台用户",
		Tags: []string{"PostgreSQL"}, Summary: "后台同步测试", Priority: "常规", Payload: payload,
	}, false)
	if err != nil {
		t.Fatalf("create admin content: %v", err)
	}
	record, err = handler.syncContentRecord(ctx, record)
	if err != nil {
		t.Fatalf("sync admin content: %v", err)
	}
	var syncedPostID int64
	if err := json.Unmarshal(record.Payload, &map[string]any{}); err != nil {
		t.Fatalf("record payload is invalid JSON: %v", err)
	}
	if err := handler.queryRow(ctx, `SELECT id FROM posts WHERE title = ? AND deleted_at IS NULL`, record.Title).Scan(&syncedPostID); err != nil {
		t.Fatalf("read synced post: %v", err)
	}
	if syncedPostID <= 0 {
		t.Fatal("synced post ID was not assigned")
	}

	record.Title = "后台 PostgreSQL 内容更新"
	updated, err := handler.upsertContent(ctx, AdminContentInput{
		ID: record.ID, Module: record.Module, Title: record.Title, Type: record.Type,
		Status: record.Status, Scope: record.Scope, Owner: record.Owner, Tags: record.Tags,
		Summary: record.Summary, URL: record.URL, Priority: record.Priority, SortOrder: record.SortOrder, Payload: record.Payload,
	}, true)
	if err != nil {
		t.Fatalf("update admin content: %v", err)
	}
	if _, err := handler.syncContentRecord(ctx, updated); err != nil {
		t.Fatalf("resync admin content: %v", err)
	}
	items, err := handler.listContentRecords(ctx, false)
	if err != nil || len(items) == 0 {
		t.Fatalf("list admin content: items=%d err=%v", len(items), err)
	}
	if err := handler.softDeleteSyncedPost(ctx, updated.Payload); err != nil {
		t.Fatalf("soft delete synced post: %v", err)
	}
}
