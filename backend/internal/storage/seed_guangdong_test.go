package storage

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"subject-choice-forum/backend/internal/config"

	"golang.org/x/crypto/bcrypt"
)

func TestSeedGuangdongPhaseOneIsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.Config{
		SQLitePath:     filepath.Join(tempDir, "soulcourse.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	}

	for attempt := 0; attempt < 2; attempt++ {
		db, err := NewSQLiteDB(cfg)
		if err != nil {
			t.Fatalf("open database on attempt %d: %v", attempt+1, err)
		}
		var users, sources, adminPosts int
		if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE email LIKE '%@soulcourse.cn'`).Scan(&users); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM content_sources`).Scan(&sources); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM admin_content_records WHERE id LIKE 'xhs-%' AND deleted_at IS NULL`).Scan(&adminPosts); err != nil {
			t.Fatal(err)
		}
		if users != 12 || sources != 80 || adminPosts != 80 {
			t.Fatalf("unexpected seed counts: users=%d sources=%d adminPosts=%d", users, sources, adminPosts)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}

	db, err := NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE email = 'yuexuan01@soulcourse.cn'`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("12345678")); err != nil {
		t.Fatalf("seed password does not match: %v", err)
	}
	var rawImages string
	if err := db.QueryRow(`SELECT image_urls FROM posts WHERE title = '深圳一模到高考：语文复盘比刷量更重要'`).Scan(&rawImages); err != nil {
		t.Fatal(err)
	}
	var images []string
	if err := json.Unmarshal([]byte(rawImages), &images); err != nil {
		t.Fatal(err)
	}
	if len(images) != 3 {
		t.Fatalf("expected all source images, got %d", len(images))
	}
	var author string
	var userID any
	if err := db.QueryRow(`
		SELECT p.author_name, p.user_id
		FROM posts p JOIN content_sources cs ON cs.post_id = p.id
		WHERE cs.source_note_id = '66a8084b0000000005020a81'
	`).Scan(&author, &userID); err != nil {
		t.Fatal(err)
	}
	if author != "情绪病Zzz" || userID != nil {
		t.Fatalf("unexpected source attribution: author=%q userID=%v", author, userID)
	}
	var adminOwner, adminPayload string
	if err := db.QueryRow(`SELECT owner, payload FROM admin_content_records WHERE id = 'xhs-66a8084b0000000005020a81'`).Scan(&adminOwner, &adminPayload); err != nil {
		t.Fatal(err)
	}
	if adminOwner != "情绪病Zzz" || !json.Valid([]byte(adminPayload)) {
		t.Fatalf("unexpected source admin record: owner=%q payload=%q", adminOwner, adminPayload)
	}
}
