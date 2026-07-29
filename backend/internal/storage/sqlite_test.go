package storage

import (
	"encoding/json"
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
		SET title = '2026届各组合专业覆盖率汇总',
		    content = '基于多省教育考试院公开数据整理。',
		    tags = '["数据建议","专业覆盖率"]',
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
	if count != 7 || total != 16071 || missingSources != 0 {
		t.Fatalf("unexpected official insight dataset: count=%d total=%d missingSources=%d", count, total, missingSources)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM subject_insights`).Scan(&allRows); err != nil {
		t.Fatal(err)
	}
	if allRows != 7 {
		t.Fatalf("migration must not duplicate official insights: got %d rows", allRows)
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
		  AND title = '广东2025专科征集志愿：16071个计划的选科要求'
		  AND province = '广东'
		  AND likes_count = 0 AND comments_count = 0 AND favorites_count = 0
	`).Scan(&migratedPostCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM posts WHERE title = '2026届各组合专业覆盖率汇总'`).Scan(&legacyPostCount); err != nil {
		t.Fatal(err)
	}
	if migratedPostCount != 1 || legacyPostCount != 0 {
		t.Fatalf("legacy insight post was not replaced exactly once: migrated=%d legacy=%d", migratedPostCount, legacyPostCount)
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
