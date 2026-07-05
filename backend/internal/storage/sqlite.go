package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"subject-choice-forum/backend/internal/config"

	_ "modernc.org/sqlite"
)

func NewSQLiteDB(cfg config.Config) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.MediaUploadDir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", cfg.SQLitePath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	for _, statement := range []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
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
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique
			ON users (lower(email))
			WHERE deleted_at IS NULL AND email IS NOT NULL;`,
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
		`CREATE TABLE IF NOT EXISTS subject_insights (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			combination TEXT NOT NULL UNIQUE,
			trend TEXT NOT NULL,
			heat INTEGER NOT NULL,
			match_rate REAL NOT NULL,
			advice TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
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
		`CREATE TABLE IF NOT EXISTS topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
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
			created_at TEXT NOT NULL
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
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := sqliteNow()
	seeds := []string{
		fmt.Sprintf(`INSERT INTO posts
			(author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province, likes_count, comments_count, favorites_count, created_at, updated_at)
			VALUES
			('小周同学', 'student', '物化生适合目标不太明确的人吗？', '我现在数学和物理还可以，化学中上，生物背诵压力能接受。想听听大家对物化生后续专业覆盖和学习强度的真实感受。', '[]', '["物化生","专业覆盖"]', 'physics', '["chemistry","biology"]', 'question', '高一', '浙江', 128, 3, 42, '%[1]s', '%[1]s'),
			('林妈妈', 'parent', '孩子想选史政地，家长应该怎么判断风险？', '孩子文科表达不错，但我们担心专业选择变窄。想请教史政地在赋分和未来专业方向上要提前注意什么。', '[]', '["史政地","风险核对"]', 'history', '["politics","geography"]', 'question', '高一', '山东', 96, 2, 31, '%[1]s', '%[1]s'),
			('陈老师', 'teacher', '从最近三届学生看物化地的优劣势', '物化地通常适合物理基础稳、空间理解强、但不想承受生物记忆量的学生。它的优势是工科覆盖较好，地理赋分在部分地区也较友好。', '[]', '["物化地","工科"]', 'physics', '["chemistry","geography"]', 'experience', '高一', '广东', 212, 4, 88, '%[1]s', '%[1]s'),
			('选科研究所', 'counselor', '2026届各组合专业覆盖率汇总', '基于多省教育考试院公开数据整理。选科前一定要把目标专业组、学校层次、赋分规则一起看，不要只看一个覆盖率数字。', '[]', '["数据建议","专业覆盖率"]', 'physics', '["chemistry","biology"]', 'data', '高一', '全国', 1100, 2, 480, '%[1]s', '%[1]s');`, now),
		fmt.Sprintf(`INSERT INTO comments (post_id, author, role, content, created_at)
			VALUES
			(1, '一只铅笔', 'student', '我也是物化生，最大感受是节奏很满，但专业覆盖确实安心。', '%[1]s'),
			(1, '王老师', 'teacher', '可以先看校内排名稳定性，不要只看一次月考。', '%[1]s'),
			(2, '周顾问', 'counselor', '建议先列出不能报考的专业清单，再判断能否接受。', '%[1]s'),
			(3, '高一新生', 'student', '感谢分享，终于看到不是只讲覆盖率的经验。', '%[1]s');`, now),
		fmt.Sprintf(`INSERT INTO subject_insights (combination, trend, heat, match_rate, advice, details, updated_at)
			VALUES
			('物理 + 化学 + 生物', '专业覆盖高，学习强度高', 96, 91.5, '适合数理基础稳定、能承受连续刷题和记忆任务的学生。', '物化生通常拥有较高专业覆盖度，但三科都需要持续投入。', '%[1]s'),
			('物理 + 化学 + 地理', '工科友好，地理赋分需看省份', 88, 84.2, '适合物理化学较稳且喜欢图表、空间分析的学生。', '适合物理化学基础较稳，但不希望承担大量生物记忆任务的学生。', '%[1]s'),
			('物理 + 生物 + 地理', '压力相对均衡，专业覆盖中高', 74, 78.4, '适合想保留理工方向但化学压力较大的学生。', '选择前应重点核对目标专业组选科要求。', '%[1]s'),
			('历史 + 政治 + 地理', '人文社科清晰，专业边界明确', 81, 73.1, '适合表达、记忆、材料分析能力强的学生。', '政治和地理都不是短期背诵型科目。', '%[1]s');`, now),
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
			('advice-note-audit-major', 'advice', '先列不能报考清单，再谈兴趣', '精选建议', '已上架', '全国', '升学规划组', '["专业限制","家长沟通"]', '把硬性选科限制先排清，再讨论兴趣、学习强度和长期目标。', '/advice/audit-major-fit', '常规', 10, '{"cover":"blue"}', '%[1]s', '%[1]s'),
			('insight-physics-chemistry-biology', 'insights', '物理 + 化学 + 生物', '组合趋势', '已上架', '全国', '数据运营', '["热度 96","覆盖 91.5%%"]', '专业覆盖高，学习强度高。', '/insights/1', '常规', 10, '{"heat":96,"matchRate":91.5}', '%[1]s', '%[1]s'),
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
	return nil
}

func sqliteNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
