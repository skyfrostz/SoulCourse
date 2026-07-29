package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/storage"
)

func TestAdminPostSyncPreservesImmutableOwnerAndPublicationState(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.Config{
		SQLitePath:     filepath.Join(tempDir, "admin-post.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := nowString()
	result, err := db.Exec(`
		INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at)
		VALUES ('admin-import-owner@example.com', 'hash', '真实账号昵称', 'student', '广东', '高一', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	handler := NewAdminHandler(cfg, nil, db)
	payload := json.RawMessage(fmt.Sprintf(`{
		"createdByUserId":"%d",
		"content":"通过后台导入但归属于真实账号的帖子。",
		"track":"physics",
		"electives":["chemistry","biology"],
		"category":"experience",
		"grade":"高一",
		"province":"广东"
	}`, userID))
	record, err := handler.upsertContent(context.Background(), AdminContentInput{
		ID: "admin-import-owned-post", Module: "posts", Title: "后台导入的账号帖子", Type: "经验帖",
		Status: "草稿", Scope: "广东", Owner: "仅用于展示的作者名", Payload: payload,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	record, err = handler.syncContentRecord(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	postID := payloadInt64(decodePayloadMap(record.Payload), "postId")
	if postID == 0 {
		t.Fatal("synced post id was not persisted")
	}
	assertSyncedPostState(t, db, postID, userID, "仅用于展示的作者名", false)

	if _, err := db.Exec(`UPDATE admin_content_records SET status = '已上架' WHERE id = ?`, record.ID); err != nil {
		t.Fatal(err)
	}
	record, err = handler.getContentByID(context.Background(), record.ID)
	if err != nil {
		t.Fatal(err)
	}
	record, err = handler.syncContentRecord(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	assertSyncedPostState(t, db, postID, userID, "仅用于展示的作者名", true)

	result, err = db.Exec(`
		INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at)
		VALUES ('other-owner@example.com', 'hash', '另一个账号', 'student', '广东', '高一', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	otherUserID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	payloadMap := decodePayloadMap(record.Payload)
	payloadMap["createdByUserId"] = fmt.Sprintf("%d", otherUserID)
	if _, err := db.Exec(`UPDATE admin_content_records SET payload = ? WHERE id = ?`, marshalJSON(payloadMap), record.ID); err != nil {
		t.Fatal(err)
	}
	record, err = handler.getContentByID(context.Background(), record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler.syncContentRecord(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	assertSyncedPostState(t, db, postID, userID, "仅用于展示的作者名", true)
}

func assertSyncedPostState(t *testing.T, db *sql.DB, postID int64, userID int64, displayName string, public bool) {
	t.Helper()
	var actualUserID int64
	var authorName string
	var deletedAt sql.NullString
	if err := db.QueryRow(`SELECT user_id, author_name, deleted_at FROM posts WHERE id = ?`, postID).Scan(&actualUserID, &authorName, &deletedAt); err != nil {
		t.Fatal(err)
	}
	if actualUserID != userID || authorName != displayName {
		t.Fatalf("post ownership/display mismatch: user=%d author=%q", actualUserID, authorName)
	}
	if public == deletedAt.Valid {
		t.Fatalf("unexpected publication state: public=%v deletedAt=%#v", public, deletedAt)
	}
}
