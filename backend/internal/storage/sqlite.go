package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"

	_ "modernc.org/sqlite"
)

func NewSQLiteDB(cfg config.Config) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0750); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.MediaUploadDir, 0750); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", cfg.SQLitePath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	for _, statement := range []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA temp_store = MEMORY;",
		"PRAGMA cache_size = -4096;",
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := initSQLiteSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func initSQLiteSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT,
			password_hash TEXT,
			nickname TEXT NOT NULL,
			role TEXT NOT NULL,
			province TEXT NOT NULL DEFAULT '',
			grade TEXT NOT NULL DEFAULT '',
			is_shadow INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			banned_at TEXT,
			banned_reason TEXT NOT NULL DEFAULT '',
			deleted_at TEXT
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique
			ON users (lower(email))
			WHERE deleted_at IS NULL AND email IS NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			revoked_at TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_active
			ON auth_sessions (user_id, expires_at)
			WHERE revoked_at IS NULL;`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			author_name TEXT NOT NULL,
			author_role TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			image_urls TEXT NOT NULL DEFAULT '[]',
			tags TEXT NOT NULL DEFAULT '[]',
			track TEXT NOT NULL,
			electives TEXT NOT NULL DEFAULT '[]',
			category TEXT NOT NULL,
			grade TEXT NOT NULL,
			province TEXT NOT NULL,
			likes_count INTEGER NOT NULL DEFAULT 0,
			comments_count INTEGER NOT NULL DEFAULT 0,
			favorites_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			user_id INTEGER,
			author TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			deleted_at TEXT,
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS content_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reporter_user_id INTEGER NOT NULL,
			target_type TEXT NOT NULL,
			target_id INTEGER NOT NULL,
			reason TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',
			resolution_note TEXT NOT NULL DEFAULT '',
			resolved_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(reporter_user_id, target_type, target_id),
			FOREIGN KEY(reporter_user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS subject_insights (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			combination TEXT NOT NULL UNIQUE,
			trend TEXT NOT NULL,
			heat INTEGER NOT NULL,
			match_rate REAL NOT NULL,
			advice TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
			metric_type TEXT NOT NULL DEFAULT '',
			unit TEXT NOT NULL DEFAULT '',
			province TEXT NOT NULL DEFAULT '',
			data_year INTEGER NOT NULL DEFAULT 0,
			source_name TEXT NOT NULL DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL DEFAULT '',
			sample_size INTEGER NOT NULL DEFAULT 0,
			captured_at TEXT NOT NULL DEFAULT '',
			methodology TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS post_likes (
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (user_id, post_id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS post_favorites (
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (user_id, post_id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS follows (
			follower_id INTEGER NOT NULL,
			author_name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (follower_id, author_name),
			FOREIGN KEY(follower_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS user_profiles (
			user_id INTEGER PRIMARY KEY,
			bio TEXT NOT NULL DEFAULT '',
			choice_profile TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			recipient_user_id INTEGER NOT NULL,
			actor_user_id INTEGER,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			target_url TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			read_at TEXT,
			FOREIGN KEY(recipient_user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(actor_user_id) REFERENCES users(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS direct_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_user_id INTEGER NOT NULL,
			recipient_user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			read_at TEXT,
			FOREIGN KEY(sender_user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(recipient_user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			topic_tag TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL,
			summary TEXT NOT NULL,
			views_count INTEGER NOT NULL DEFAULT 0,
			posts_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS topic_posts (
			topic_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			PRIMARY KEY (topic_id, post_id),
			FOREIGN KEY(topic_id) REFERENCES topics(id) ON DELETE CASCADE,
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS email_verification_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			code_hash TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			used_at TEXT,
			failed_attempts INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS email_verification_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			client_ip TEXT NOT NULL,
			created_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS upload_assets (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			asset_key TEXT NOT NULL UNIQUE,
			file_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			ext TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			completed_at TEXT,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS admin_content_records (
			id TEXT PRIMARY KEY,
			module TEXT NOT NULL,
			title TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '草稿',
			scope TEXT NOT NULL DEFAULT '全国',
			owner TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '[]',
			summary TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL DEFAULT '常规',
			sort_order INTEGER NOT NULL DEFAULT 0,
			payload TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS admin_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			record_id TEXT,
			module TEXT,
			detail TEXT NOT NULL DEFAULT '',
			actor TEXT NOT NULL DEFAULT 'admin',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS content_sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL UNIQUE,
			source_platform TEXT NOT NULL,
			source_url TEXT NOT NULL UNIQUE,
			 source_note_id TEXT NOT NULL DEFAULT '',
			 source_title TEXT NOT NULL DEFAULT '',
			 source_author TEXT NOT NULL DEFAULT '',
			 source_avatar_url TEXT NOT NULL DEFAULT '',
			 source_likes INTEGER NOT NULL DEFAULT 0,
			source_comments INTEGER NOT NULL DEFAULT 0,
			source_favorites INTEGER NOT NULL DEFAULT 0,
			source_format TEXT NOT NULL DEFAULT '图文',
			transformation_note TEXT NOT NULL DEFAULT '',
			captured_at TEXT NOT NULL,
			FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS app_migrations (
			name TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_posts_feed
			ON posts (deleted_at, track, category, province, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_posts_author
			ON posts (author_name, deleted_at);`,
		`CREATE INDEX IF NOT EXISTS idx_posts_user
			ON posts (user_id, deleted_at, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post
			ON comments (post_id, deleted_at, created_at ASC);`,
		`CREATE INDEX IF NOT EXISTS idx_content_reports_status
			ON content_reports (status, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_recipient
			ON notifications (recipient_user_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_direct_messages_sender
			ON direct_messages (sender_user_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_direct_messages_recipient
			ON direct_messages (recipient_user_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_direct_messages_pair_sender
			ON direct_messages (sender_user_id, recipient_user_id, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_direct_messages_pair_recipient
			ON direct_messages (recipient_user_id, sender_user_id, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_email_verification_lookup
			ON email_verification_codes (email, used_at, expires_at, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_email_verification_attempts_email
			ON email_verification_attempts (email, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_email_verification_attempts_ip
			ON email_verification_attempts (client_ip, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_admin_content_listing
			ON admin_content_records (deleted_at, module, status, sort_order, updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_admin_audit_created
			ON admin_audit_logs (created_at DESC);`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	if err := ensureSQLiteColumn(db, "users", "is_shadow", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(db, "users", "banned_at", "TEXT"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(db, "users", "banned_reason", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_shadow_nickname
		ON users (nickname) WHERE is_shadow = 1 AND deleted_at IS NULL`); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(db, "content_sources", "source_avatar_url", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(db, "email_verification_codes", "failed_attempts", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := ensureSQLiteColumn(db, "topics", "topic_tag", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	for column, definition := range map[string]string{
		"metric_type": "TEXT NOT NULL DEFAULT ''",
		"unit":        "TEXT NOT NULL DEFAULT ''",
		"province":    "TEXT NOT NULL DEFAULT ''",
		"data_year":   "INTEGER NOT NULL DEFAULT 0",
		"source_name": "TEXT NOT NULL DEFAULT ''",
		"source_url":  "TEXT NOT NULL DEFAULT ''",
		"scope":       "TEXT NOT NULL DEFAULT ''",
		"sample_size": "INTEGER NOT NULL DEFAULT 0",
		"captured_at": "TEXT NOT NULL DEFAULT ''",
		"methodology": "TEXT NOT NULL DEFAULT ''",
	} {
		if err := ensureSQLiteColumn(db, "subject_insights", column, definition); err != nil {
			return err
		}
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		now := sqliteNow()
		seeds := []string{
			fmt.Sprintf(`INSERT INTO posts
			(author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province, likes_count, comments_count, favorites_count, created_at, updated_at)
			VALUES
			('小周同学', 'student', '物化生适合目标不太明确的人吗？', '我现在数学和物理还可以，化学中上，生物背诵压力能接受。想听听大家对物化生后续专业覆盖和学习强度的真实感受。', '[]', '["物化生","专业覆盖"]', 'physics', '["chemistry","biology"]', 'question', '高一', '浙江', 128, 3, 42, '%[1]s', '%[1]s'),
			('林妈妈', 'parent', '孩子想选史政地，家长应该怎么判断风险？', '孩子文科表达不错，但我们担心专业选择变窄。想请教史政地在赋分和未来专业方向上要提前注意什么。', '[]', '["史政地","风险核对"]', 'history', '["politics","geography"]', 'question', '高一', '山东', 96, 2, 31, '%[1]s', '%[1]s'),
			('陈老师', 'teacher', '从最近三届学生看物化地的优劣势', '物化地通常适合物理基础稳、空间理解强、但不想承受生物记忆量的学生。它的优势是工科覆盖较好，地理赋分在部分地区也较友好。', '[]', '["物化地","工科"]', 'physics', '["chemistry","geography"]', 'experience', '高一', '广东', 212, 4, 88, '%[1]s', '%[1]s'),
			('选科研究所', 'counselor', '广东2025本科征集志愿：27681个计划的选科要求', '按广东省教育考试院2025年本科批次征集志愿普通类物理、历史计划表逐行汇总：物理类569条、历史类280条，共849条专业计划记录、27681个计划数。该统计是本科征集志愿招生计划要求，不是考生选科人数或录取概率。', '[]', '["数据建议","广东考试院","本科招生计划"]', 'physics', '["chemistry","biology"]', 'data', '高一', '广东', 0, 0, 0, '%[1]s', '%[1]s');`, now),
			fmt.Sprintf(`INSERT INTO comments (post_id, author, role, content, created_at)
			VALUES
			(1, '一只铅笔', 'student', '我也是物化生，最大感受是节奏很满，但专业覆盖确实安心。', '%[1]s'),
			(1, '王老师', 'teacher', '可以先看校内排名稳定性，不要只看一次月考。', '%[1]s'),
			(2, '周顾问', 'counselor', '建议先列出不能报考的专业清单，再判断能否接受。', '%[1]s'),
			(3, '高一新生', 'student', '感谢分享，终于看到不是只讲覆盖率的经验。', '%[1]s');`, now),
			fmt.Sprintf(`INSERT INTO topics (slug, title, summary, views_count, posts_count, created_at)
			VALUES
			('physics-track-how-to-choose', '物理方向组合怎么选', '围绕物理方向下物化生、物化地、物生地等组合的专业覆盖、学习强度和赋分风险展开讨论。', 7600, 3, '%[1]s'),
			('history-track-careers', '历史方向就业前景', '讨论史政地等历史方向组合与专业选择、就业想象之间的真实关系。', 6200, 1, '%[1]s'),
			('is-chemistry-important', '化学到底有多重要', '集中讨论化学在专业限制、学习难度和长期提分中的作用。', 5100, 3, '%[1]s'),
			('grade-eleven-timeline', '高二选科时间线', '汇总选科后分班、适应、补弱和阶段复盘的时间安排。', 4300, 4, '%[1]s'),
			('after-selection-score-up', '选科后如何提分', '分享选科完成后各科提分方法、错题管理和复盘节奏。', 3800, 1, '%[1]s');`, now),
			`INSERT INTO topic_posts (topic_id, post_id) VALUES
			(1, 1), (1, 3), (1, 4),
			(2, 2),
			(3, 1), (3, 3), (3, 4),
			(4, 1), (4, 2), (4, 3), (4, 4),
			(5, 3);`,
			fmt.Sprintf(`INSERT INTO admin_content_records
			(id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload, created_at, updated_at)
			VALUES
			('category-experience', 'categories', '经验帖', '帖子分类', '已上架', '全站', '内容运营', '["首页可见","社区内容"]', '学生、老师、规划师的真实经验分享。', '', '常规', 10, '{"frontendRoute":"/","category":"experience"}', '%[1]s', '%[1]s'),
			('nav-requirements', 'categories', '选科查询', '工具入口', '已上架', '全站', '产品运营', '["专业要求","工具"]', '按专业和组合查询选科约束。', '/requirements', '高', 20, '{"frontendRoute":"/requirements"}', '%[1]s', '%[1]s'),
			('policy-national-major-subject', 'policies', '普通高校本科专业选考科目要求汇总', '官方来源', '已上架', '全国', '阳光高考/学信网', '["选考要求","专业目录","官方来源"]', '用于核对不同省份、院校、专业组的选考科目限制。', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202201/20220105/2155365943.html', '高', 20, '{"publisher":"阳光高考/学信网","reviewCycle":"季度"}', '%[1]s', '%[1]s'),
			('requirement-computer-science', 'requirements', '计算机科学与技术', '物理+化学强约束', '已上架', '全国', '专业要求库', '["物理","化学","计算机"]', '计算机类专业在多数新高考省份对物理、化学有较强要求。', '/requirements/计算机科学与技术', '高', 20, '{"category":"计算机与电子信息"}', '%[1]s', '%[1]s'),
			('post-existing-1', 'posts', '物化生适合目标不太明确的人吗？', '提问', '已上架', '浙江', '小周同学', '["物化生","专业覆盖"]', '我现在数学和物理还可以，化学中上，生物背诵压力能接受。', '/posts/1', '常规', 101, '{"postId":"1","content":"我现在数学和物理还可以，化学中上，生物背诵压力能接受。想听听大家对物化生后续专业覆盖和学习强度的真实感受。","track":"physics","electives":["chemistry","biology"],"category":"question","grade":"高一","province":"浙江","imageUrls":[]}', '%[1]s', '%[1]s'),
			('post-existing-2', 'posts', '孩子想选史政地，家长应该怎么判断风险？', '家长提问', '待审核', '山东', '林妈妈', '["史政地","风险核对"]', '孩子文科表达不错，但我们担心专业选择变窄。', '/posts/2', '中', 102, '{"postId":"2","content":"孩子文科表达不错，但我们担心专业选择变窄。想请教史政地在赋分和未来专业方向上要提前注意什么。","track":"history","electives":["politics","geography"],"category":"question","grade":"高一","province":"山东","imageUrls":[]}', '%[1]s', '%[1]s');`, now),
			fmt.Sprintf(`INSERT INTO admin_audit_logs (action, record_id, module, detail, actor, created_at)
			VALUES ('bootstrap', '', 'system', 'SQLite 初始内容库已完成导入', 'system', '%[1]s');`, now),
		}
		for _, statement := range seeds {
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("seed sqlite database: %w", err)
			}
		}
	}
	if err := migrateOfficialSubjectInsights(db); err != nil {
		return err
	}
	if err := backfillTopicTags(db); err != nil {
		return err
	}
	if err := migrateLegacyTopicPostTags(db); err != nil {
		return err
	}
	if err := backfillSubjectPostTags(db); err != nil {
		return err
	}
	if err := backfillPostUserOwnership(db); err != nil {
		return err
	}
	if err := hideNonPublicAdminPosts(db); err != nil {
		return err
	}
	return nil
}

func migrateOfficialSubjectInsights(db *sql.DB) error {
	const metricType = "admission_plan_requirement_count"
	now := sqliteNow()
	if _, err := db.Exec(`DELETE FROM admin_content_records WHERE id IN ('advice-note-audit-major', 'insight-physics-chemistry-biology')`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		UPDATE posts
		SET title = '广东2025本科征集志愿：27681个计划的选科要求',
		    content = '按广东省教育考试院2025年本科批次征集志愿普通类物理、历史计划表逐行汇总：物理类569条、历史类280条，共849条专业计划记录、27681个计划数。该统计是本科征集志愿招生计划要求，不是考生选科人数或录取概率。',
		    tags = '["数据建议","广东考试院","本科招生计划"]', province = '广东',
		    likes_count = 0, comments_count = 0, favorites_count = 0, updated_at = ?
		WHERE author_name = '选科研究所'
		  AND title IN ('2026届各组合专业覆盖率汇总', '广东2025专科征集志愿：16071个计划的选科要求', '广东2025本科征集志愿：27681个计划的选科要求')
	`, now); err != nil {
		return err
	}
	var rowCount, total int
	if err := db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(heat), 0) FROM subject_insights WHERE metric_type = ? AND province = '广东' AND data_year = 2025`, metricType).Scan(&rowCount, &total); err != nil {
		return err
	}
	if rowCount == 8 && total == 27681 {
		return nil
	}
	type officialInsight struct {
		combination string
		trend       string
		value       int
		share       float64
		sourceURL   string
	}
	items := []officialInsight{
		{combination: "物理 + 再选不限", trend: "再选科目不限", value: 10810, share: 39.0521, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
		{combination: "物理 + 化学", trend: "再选科目要求化学", value: 8446, share: 30.5119, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
		{combination: "历史 + 再选不限", trend: "再选科目不限", value: 7928, share: 28.6406, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
		{combination: "历史 + 政治", trend: "再选科目要求政治", value: 324, share: 1.1705, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
		{combination: "物理 + 生物", trend: "再选科目要求生物", value: 69, share: 0.2493, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
		{combination: "历史 + 生物", trend: "再选科目要求生物", value: 57, share: 0.2059, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
		{combination: "历史 + 化学", trend: "再选科目要求化学", value: 45, share: 0.1626, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
		{combination: "物理 + 化学 + 生物", trend: "再选科目要求化学和生物", value: 2, share: 0.0072, sourceURL: "https://eea.gd.gov.cn/attachment/0/586/586507/4749214.zip"},
	}
	transaction, err := db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if _, err := transaction.Exec(`DELETE FROM subject_insights`); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := transaction.Exec(`
			INSERT INTO subject_insights
			(combination, trend, heat, match_rate, advice, details, metric_type, unit, province, data_year,
			 source_name, source_url, scope, sample_size, captured_at, methodology, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, '招生计划数', '广东', 2025, '广东省教育考试院', ?, ?, 27681, ?, ?, ?)
		`, item.combination, item.trend, item.value, item.share,
			"该指标只反映招生计划的选科要求，不代表考生选科人数、录取概率或组合优劣。",
			fmt.Sprintf("该类别共有 %d 个招生计划，占本数据集 %.4f%%。", item.value, item.share),
			metricType, item.sourceURL,
			"广东省2025年普通高校招生本科院校征集志愿（普通类历史、物理）",
			now,
			"逐行读取物理类569条、历史类280条官方本科计划记录，按首选科目和再选科目要求汇总27681个计划数。",
			now,
		); err != nil {
			return err
		}
	}
	return transaction.Commit()
}

func backfillUnownedPostAuthors(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT author_name, author_role, province, grade
		FROM posts
		WHERE user_id IS NULL AND TRIM(author_name) <> ''
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return err
	}
	type authorIdentity struct {
		name     string
		role     string
		province string
		grade    string
	}
	authors := make([]authorIdentity, 0)
	seen := make(map[string]bool)
	for rows.Next() {
		var author authorIdentity
		if err := rows.Scan(&author.name, &author.role, &author.province, &author.grade); err != nil {
			rows.Close()
			return err
		}
		author.name = strings.TrimSpace(author.name)
		if author.name == "" || seen[author.name] {
			continue
		}
		seen[author.name] = true
		authors = append(authors, author)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	transaction, err := db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	now := sqliteNow()
	for _, author := range authors {
		var userID int64
		err := transaction.QueryRow(`
			SELECT id FROM users
			WHERE nickname = ? AND is_shadow = 1 AND deleted_at IS NULL
			ORDER BY id ASC LIMIT 1
		`, author.name).Scan(&userID)
		if errors.Is(err, sql.ErrNoRows) {
			result, insertErr := transaction.Exec(`
				INSERT INTO users (email, password_hash, nickname, role, province, grade, is_shadow, created_at, updated_at)
				VALUES (NULL, NULL, ?, ?, ?, ?, 1, ?, ?)
			`, author.name, author.role, author.province, author.grade, now, now)
			if insertErr != nil {
				return insertErr
			}
			userID, err = result.LastInsertId()
		}
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(`
			UPDATE posts SET user_id = ?
			WHERE user_id IS NULL AND author_name = ?
		`, userID, author.name); err != nil {
			return err
		}
	}
	return transaction.Commit()
}

func backfillTopicTags(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, slug FROM topics WHERE topic_tag = ''`)
	if err != nil {
		return err
	}
	type topicUpdate struct {
		id  int64
		tag string
	}
	updates := make([]topicUpdate, 0)
	for rows.Next() {
		var id int64
		var slug string
		if err := rows.Scan(&id, &slug); err != nil {
			rows.Close()
			return err
		}
		if tag, ok := domain.TopicTagForSlug(slug); ok {
			updates = append(updates, topicUpdate{id: id, tag: tag})
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, update := range updates {
		if _, err := db.Exec(`UPDATE topics SET topic_tag = ? WHERE id = ? AND topic_tag = ''`, update.tag, update.id); err != nil {
			return err
		}
	}
	return nil
}

func migrateLegacyTopicPostTags(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT p.id, p.tags, t.topic_tag
		FROM topic_posts tp
		JOIN topics t ON t.id = tp.topic_id
		JOIN posts p ON p.id = tp.post_id
		WHERE t.topic_tag <> ''
		ORDER BY p.id, t.id
	`)
	if err != nil {
		return err
	}
	tagsByPost := make(map[int64][]string)
	for rows.Next() {
		var postID int64
		var rawTags, topicTag string
		if err := rows.Scan(&postID, &rawTags, &topicTag); err != nil {
			rows.Close()
			return err
		}
		tags, exists := tagsByPost[postID]
		if !exists {
			if err := json.Unmarshal([]byte(rawTags), &tags); err != nil {
				rows.Close()
				return fmt.Errorf("migrate topic tags for post %d: %w", postID, err)
			}
		}
		if !containsString(tags, topicTag) {
			tags = append(tags, topicTag)
		}
		tagsByPost[postID] = tags
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	transaction, err := db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	for postID, tags := range tagsByPost {
		encoded, err := json.Marshal(tags)
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(`UPDATE posts SET tags = ? WHERE id = ?`, string(encoded), postID); err != nil {
			return err
		}
	}
	return transaction.Commit()
}

func backfillSubjectPostTags(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, tags, track, electives FROM posts WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	tagsByPost := make(map[int64][]string)
	for rows.Next() {
		var postID int64
		var rawTags, rawTrack, rawElectives string
		if err := rows.Scan(&postID, &rawTags, &rawTrack, &rawElectives); err != nil {
			rows.Close()
			return err
		}
		var tags []string
		var electives []domain.Subject
		if err := json.Unmarshal([]byte(rawTags), &tags); err != nil {
			rows.Close()
			return fmt.Errorf("migrate subject tags for post %d: %w", postID, err)
		}
		if err := json.Unmarshal([]byte(rawElectives), &electives); err != nil {
			rows.Close()
			return fmt.Errorf("parse electives for post %d: %w", postID, err)
		}
		tag, ok := domain.SubjectTagForChoice(domain.SubjectTrack(rawTrack), electives)
		if ok && !containsString(tags, tag) {
			tagsByPost[postID] = append(tags, tag)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	transaction, err := db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	for postID, tags := range tagsByPost {
		encoded, err := json.Marshal(tags)
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(`UPDATE posts SET tags = ? WHERE id = ?`, string(encoded), postID); err != nil {
			return err
		}
	}
	return transaction.Commit()
}

func backfillPostUserOwnership(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT payload
		FROM admin_content_records
		WHERE module = 'posts' AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return err
	}
	ownersByPost := make(map[int64]int64)
	for rows.Next() {
		var rawPayload string
		if err := rows.Scan(&rawPayload); err != nil {
			rows.Close()
			return err
		}
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
			rows.Close()
			return fmt.Errorf("migrate post ownership: %w", err)
		}
		postID := migrationPayloadID(payload, "postId")
		userID := migrationPayloadID(payload, "createdByUserId")
		if userID == 0 {
			userID = migrationPayloadID(payload, "userId")
		}
		if postID == 0 || userID == 0 {
			continue
		}
		if existing, ok := ownersByPost[postID]; ok && existing != userID {
			ownersByPost[postID] = 0
			continue
		}
		ownersByPost[postID] = userID
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	transaction, err := db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	for postID, userID := range ownersByPost {
		if userID == 0 {
			continue
		}
		if _, err := transaction.Exec(`
			UPDATE posts
			SET user_id = ?
			WHERE id = ? AND user_id IS NULL
			  AND EXISTS (SELECT 1 FROM users WHERE id = ?)
		`, userID, postID, userID); err != nil {
			return err
		}
	}
	if err := transaction.Commit(); err != nil {
		return err
	}
	return backfillUnownedPostAuthors(db)
}

func migrationPayloadID(payload map[string]any, key string) int64 {
	value, ok := payload[key]
	if !ok || value == nil {
		return 0
	}
	id, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(value)), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func hideNonPublicAdminPosts(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT payload
		FROM admin_content_records
		WHERE module = 'posts' AND deleted_at IS NULL
		  AND status NOT IN ('已上架', '正常')
	`)
	if err != nil {
		return err
	}
	postIDs := make(map[int64]bool)
	for rows.Next() {
		var rawPayload string
		if err := rows.Scan(&rawPayload); err != nil {
			rows.Close()
			return err
		}
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(rawPayload), &payload); err != nil {
			rows.Close()
			return fmt.Errorf("migrate post visibility: %w", err)
		}
		if postID := migrationPayloadID(payload, "postId"); postID > 0 {
			postIDs[postID] = true
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	hiddenAt := sqliteNow()
	transaction, err := db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	for postID := range postIDs {
		if _, err := transaction.Exec(`UPDATE posts SET deleted_at = COALESCE(deleted_at, ?) WHERE id = ?`, hiddenAt, postID); err != nil {
			return err
		}
	}
	return transaction.Commit()
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func ensureSQLiteColumn(db *sql.DB, table string, column string, definition string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
	return err
}

func sqliteNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
