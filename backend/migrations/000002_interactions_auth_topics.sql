ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(120);
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS province VARCHAR(30);
ALTER TABLE users ADD COLUMN IF NOT EXISTS grade VARCHAR(20);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users (lower(email)) WHERE deleted_at IS NULL AND email IS NOT NULL;

ALTER TABLE posts ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE subject_insights ADD COLUMN IF NOT EXISTS details TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS post_likes (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, post_id)
);

CREATE TABLE IF NOT EXISTS post_favorites (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, post_id)
);

CREATE TABLE IF NOT EXISTS follows (
  follower_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  author_name VARCHAR(40) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (follower_id, author_name)
);

CREATE TABLE IF NOT EXISTS topics (
  id BIGSERIAL PRIMARY KEY,
  slug VARCHAR(80) NOT NULL UNIQUE,
  title VARCHAR(80) NOT NULL,
  summary TEXT NOT NULL,
  views_count INTEGER NOT NULL DEFAULT 0,
  posts_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS topic_posts (
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  PRIMARY KEY (topic_id, post_id)
);

INSERT INTO users (email, password_hash, nickname, role, province, grade)
VALUES
  ('demo@student.local', '$2a$10$t/JFyW/YAZz4Vj.XyAkkAeQWgPniZgyo6UtQrqs3Don7bpUBp3ZHO', '演示同学', 'student', '浙江', '高一'),
  ('demo@parent.local', '$2a$10$t/JFyW/YAZz4Vj.XyAkkAeQWgPniZgyo6UtQrqs3Don7bpUBp3ZHO', '演示家长', 'parent', '江苏', '高一')
ON CONFLICT DO NOTHING;

UPDATE subject_insights
SET details = CASE combination
  WHEN '物理 + 化学 + 生物' THEN '物化生通常拥有较高专业覆盖度，尤其适合目标仍在理工、医学、农学、生物科学等方向间摇摆的学生。它的真实门槛在于三科都需要持续投入：物理看模型，化学看体系，生物看细节记忆。建议用最近三次校内考试排名稳定性、错题修复速度和每天可投入时间共同判断。'
  WHEN '物理 + 化学 + 地理' THEN '物化地在工科方向上较友好，地理相较生物更强调图表、区域分析和综合理解。适合物理化学基础较稳，但不希望承担大量生物记忆任务的学生。需要结合本省赋分规则和学校地理教学强度判断。'
  WHEN '物理 + 生物 + 地理' THEN '物生地整体压力相对均衡，但部分化学限选专业会受到影响。适合想保留理工大方向、同时化学长期吃力的学生。选择前应重点核对目标专业组选科要求。'
  WHEN '历史 + 政治 + 地理' THEN '史政地路径清晰，适合人文社科、法学、管理、教育等方向兴趣更明确的学生。政治和地理都不是短期背诵型科目，稳定输出需要材料分析、时政积累和图表训练。'
  ELSE '这是一个相对小众组合，建议优先从目标专业限制、校内师资、个人优势科目和所在省份赋分规则四个维度做交叉判断。'
END
WHERE details = '';

INSERT INTO topics (slug, title, summary, views_count, posts_count)
VALUES
  ('physics-track-how-to-choose', '物理方向组合怎么选', '围绕物理方向下物化生、物化地、物生地等组合的专业覆盖、学习强度和赋分风险展开讨论。', 7600, 0),
  ('history-track-careers', '历史方向就业前景', '讨论史政地、史政生等历史方向组合与专业选择、就业想象之间的真实关系。', 6200, 0),
  ('is-chemistry-important', '化学到底有多重要', '集中讨论化学在专业限制、学习难度和长期提分中的作用。', 5100, 0),
  ('grade-eleven-timeline', '高二选科时间线', '汇总选科后分班、适应、补弱和阶段复盘的时间安排。', 4300, 0),
  ('after-selection-score-up', '选科后如何提分', '分享选科完成后各科提分方法、错题管理和复盘节奏。', 3800, 0)
ON CONFLICT (slug) DO UPDATE
SET title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    views_count = GREATEST(topics.views_count, EXCLUDED.views_count);

INSERT INTO topic_posts (topic_id, post_id)
SELECT t.id, p.id
FROM topics t
JOIN posts p ON
  (t.slug = 'physics-track-how-to-choose' AND p.track = 'physics')
  OR (t.slug = 'history-track-careers' AND p.track = 'history')
  OR (t.slug = 'is-chemistry-important' AND 'chemistry' = ANY(p.electives))
  OR (t.slug = 'grade-eleven-timeline' AND p.grade IN ('高一', '高二'))
  OR (t.slug = 'after-selection-score-up' AND p.category = 'experience')
ON CONFLICT DO NOTHING;

UPDATE topics
SET posts_count = counts.total
FROM (
  SELECT topic_id, COUNT(*)::integer AS total
  FROM topic_posts
  GROUP BY topic_id
) counts
WHERE topics.id = counts.topic_id;
