package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
)

func TestLegacyTopicMigrationIsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	db, err := NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "legacy.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`UPDATE posts SET tags = '["物化生","专业覆盖"]' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE topics DROP COLUMN topic_tag`); err != nil {
		t.Fatal(err)
	}
	if err := initSQLiteSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := initSQLiteSchema(db); err != nil {
		t.Fatal(err)
	}

	var topicTag string
	if err := db.QueryRow(`SELECT topic_tag FROM topics WHERE slug = 'physics-track-how-to-choose'`).Scan(&topicTag); err != nil {
		t.Fatal(err)
	}
	if topicTag != domain.TopicTagPhysicsTrack {
		t.Fatalf("unexpected topic tag: %q", topicTag)
	}

	var rawTags string
	if err := db.QueryRow(`SELECT tags FROM posts WHERE id = 1`).Scan(&rawTags); err != nil {
		t.Fatal(err)
	}
	var tags []string
	if err := json.Unmarshal([]byte(rawTags), &tags); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{domain.TopicTagPhysicsTrack, domain.TopicTagChemistry, domain.TopicTagSelectionTiming} {
		if countString(tags, expected) != 1 {
			t.Fatalf("migration must add %q exactly once: %#v", expected, tags)
		}
	}
}

func TestOfficialSubjectInsightsReplaceMockData(t *testing.T) {
	tempDir := t.TempDir()
	db, err := NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "insights.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		UPDATE posts
		SET title = '广东2025专科征集志愿：16071个计划的选科要求',
		    content = '按广东省教育考试院2025年专科批次第一次征集志愿普通类物理、历史计划表逐行汇总。',
		    tags = '["数据建议","广东考试院","招生计划"]',
		    province = '全国', likes_count = 1100, comments_count = 2, favorites_count = 480
		WHERE author_name = '选科研究所'
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM subject_insights`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO subject_insights (combination, trend, heat, match_rate, advice, details, updated_at)
		VALUES
		('物理 + 化学 + 生物', '专业覆盖高，学习强度高', 96, 91.5, 'mock', 'mock', '2026-01-01T00:00:00Z'),
		('物理 + 化学 + 地理', '工科友好，地理赋分需看省份', 88, 84.2, 'mock', 'mock', '2026-01-01T00:00:00Z'),
		('物理 + 生物 + 地理', '压力相对均衡，专业覆盖中高', 74, 78.4, 'mock', 'mock', '2026-01-01T00:00:00Z'),
		('历史 + 政治 + 地理', '人文社科清晰，专业边界明确', 81, 73.1, 'mock', 'mock', '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatal(err)
	}

	if err := initSQLiteSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := initSQLiteSchema(db); err != nil {
		t.Fatal(err)
	}

	var count, total, missingSources, allRows int
	if err := db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(heat), 0),
		       SUM(CASE WHEN source_url = '' OR source_name = '' OR captured_at = '' OR methodology = '' THEN 1 ELSE 0 END)
		FROM subject_insights
		WHERE metric_type = 'admission_plan_requirement_count' AND province = '广东' AND data_year = 2025
	`).Scan(&count, &total, &missingSources); err != nil {
		t.Fatal(err)
	}
	if count != 8 || total != 27681 || missingSources != 0 {
		t.Fatalf("unexpected official insight dataset: count=%d total=%d missingSources=%d", count, total, missingSources)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM subject_insights`).Scan(&allRows); err != nil {
		t.Fatal(err)
	}
	if allRows != 8 {
		t.Fatalf("migration must not duplicate official insights: got %d rows", allRows)
	}
	expectedPlans := map[string]int{
		"物理 + 再选不限":    10810,
		"物理 + 化学":      8446,
		"历史 + 再选不限":    7928,
		"历史 + 政治":      324,
		"物理 + 生物":      69,
		"历史 + 生物":      57,
		"历史 + 化学":      45,
		"物理 + 化学 + 生物": 2,
	}
	rows, err := db.Query(`
		SELECT combination, heat
		FROM subject_insights
		WHERE metric_type = 'admission_plan_requirement_count' AND province = '广东' AND data_year = 2025
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var combination string
		var plans int
		if err := rows.Scan(&combination, &plans); err != nil {
			t.Fatal(err)
		}
		if expected, ok := expectedPlans[combination]; !ok || plans != expected {
			t.Fatalf("unexpected undergraduate plan aggregate: %q=%d", combination, plans)
		}
		delete(expectedPlans, combination)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(expectedPlans) != 0 {
		t.Fatalf("missing undergraduate plan aggregates: %#v", expectedPlans)
	}
	var mockCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM subject_insights WHERE heat IN (96, 88, 81, 74)`).Scan(&mockCount); err != nil {
		t.Fatal(err)
	}
	if mockCount != 0 {
		t.Fatalf("legacy mock insights remain: %d", mockCount)
	}

	var migratedPostCount, legacyPostCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE author_name = '选科研究所'
		  AND title = '广东2025本科征集志愿：27681个计划的选科要求'
		  AND province = '广东'
		  AND likes_count = 0 AND comments_count = 0 AND favorites_count = 0
	`).Scan(&migratedPostCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE title IN ('2026届各组合专业覆盖率汇总', '广东2025专科征集志愿：16071个计划的选科要求')
	`).Scan(&legacyPostCount); err != nil {
		t.Fatal(err)
	}
	if migratedPostCount != 1 || legacyPostCount != 0 {
		t.Fatalf("legacy insight post was not replaced exactly once: migrated=%d legacy=%d", migratedPostCount, legacyPostCount)
	}
}

func TestPostOwnershipMigrationUsesOnlyImmutablePayloadIDs(t *testing.T) {
	tempDir := t.TempDir()
	db, err := NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "ownership.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := sqliteNow()
	result, err := db.Exec(`
		INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at)
		VALUES ('owner@example.com', 'hash', '导入作者', 'student', '广东', '高一', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	insertPost := func(title string) int64 {
		t.Helper()
		result, err := db.Exec(`
			INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, created_at, updated_at)
			VALUES ('导入作者', 'student', ?, '导入内容', 'physics', '["chemistry","biology"]', 'experience', '高一', '广东', ?, ?)
		`, title, now, now)
		if err != nil {
			t.Fatal(err)
		}
		postID, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return postID
	}
	linkedPostID := insertPost("带不可变账号 ID 的导入帖子")
	unlinkedPostID := insertPost("只有同名作者的导入帖子")
	draftPostID := insertPost("尚未公开的导入帖子")
	if _, err := db.Exec(`
		INSERT INTO admin_content_records
		(id, module, title, status, payload, created_at, updated_at)
		VALUES ('import-owned-post', 'posts', '带不可变账号 ID 的导入帖子', '已上架', ?, ?, ?)
	`, fmt.Sprintf(`{"postId":"%d","createdByUserId":"%d"}`, linkedPostID, userID), now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO admin_content_records
		(id, module, title, status, payload, created_at, updated_at)
		VALUES ('import-draft-post', 'posts', '尚未公开的导入帖子', '草稿', ?, ?, ?)
	`, fmt.Sprintf(`{"postId":"%d","createdByUserId":"%d"}`, draftPostID, userID), now, now); err != nil {
		t.Fatal(err)
	}

	if err := backfillPostUserOwnership(db); err != nil {
		t.Fatal(err)
	}
	if err := hideNonPublicAdminPosts(db); err != nil {
		t.Fatal(err)
	}
	if err := hideNonPublicAdminPosts(db); err != nil {
		t.Fatal(err)
	}
	if err := backfillPostUserOwnership(db); err != nil {
		t.Fatal(err)
	}
	var linkedOwner sql.NullInt64
	if err := db.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, linkedPostID).Scan(&linkedOwner); err != nil {
		t.Fatal(err)
	}
	if !linkedOwner.Valid || linkedOwner.Int64 != userID {
		t.Fatalf("immutable import ownership was not migrated: %#v", linkedOwner)
	}
	var unlinkedOwner sql.NullInt64
	if err := db.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, unlinkedPostID).Scan(&unlinkedOwner); err != nil {
		t.Fatal(err)
	}
	if !unlinkedOwner.Valid || unlinkedOwner.Int64 == userID {
		t.Fatalf("nickname-only import must receive a separate shadow identity: %#v", unlinkedOwner)
	}
	var shadowCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM users
		WHERE id = ? AND nickname = '导入作者' AND is_shadow = 1 AND email IS NULL AND password_hash IS NULL
	`, unlinkedOwner.Int64).Scan(&shadowCount); err != nil {
		t.Fatal(err)
	}
	if shadowCount != 1 {
		t.Fatalf("external author identity was not created idempotently: %d", shadowCount)
	}
	var draftOwner sql.NullInt64
	var draftDeletedAt sql.NullString
	if err := db.QueryRow(`SELECT user_id, deleted_at FROM posts WHERE id = ?`, draftPostID).Scan(&draftOwner, &draftDeletedAt); err != nil {
		t.Fatal(err)
	}
	if !draftOwner.Valid || draftOwner.Int64 != userID || !draftDeletedAt.Valid {
		t.Fatalf("non-public import must keep ownership but stay hidden: owner=%#v deletedAt=%#v", draftOwner, draftDeletedAt)
	}
	var unownedPosts int
	if err := db.QueryRow(`SELECT COUNT(*) FROM posts WHERE user_id IS NULL`).Scan(&unownedPosts); err != nil {
		t.Fatal(err)
	}
	if unownedPosts != 0 {
		t.Fatalf("every post must have a stable owner identity: %d unowned", unownedPosts)
	}
}

func countString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
