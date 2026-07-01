CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  nickname VARCHAR(40) NOT NULL,
  role VARCHAR(20) NOT NULL CHECK (role IN ('student', 'parent', 'teacher', 'counselor')),
  province VARCHAR(30),
  grade VARCHAR(20),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS posts (
  id BIGSERIAL PRIMARY KEY,
  author_name VARCHAR(40) NOT NULL,
  author_role VARCHAR(20) NOT NULL CHECK (author_role IN ('student', 'parent', 'teacher', 'counselor')),
  title VARCHAR(80) NOT NULL,
  content TEXT NOT NULL,
  track VARCHAR(20) NOT NULL CHECK (track IN ('physics', 'history')),
  electives TEXT[] NOT NULL,
  category VARCHAR(20) NOT NULL CHECK (category IN ('experience', 'question', 'data')),
  grade VARCHAR(20) NOT NULL,
  province VARCHAR(30) NOT NULL,
  likes_count INTEGER NOT NULL DEFAULT 0,
  comments_count INTEGER NOT NULL DEFAULT 0,
  favorites_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT posts_two_electives CHECK (array_length(electives, 1) = 2),
  CONSTRAINT posts_supported_electives CHECK (electives <@ ARRAY['chemistry', 'biology', 'politics', 'geography']::TEXT[])
);

CREATE TABLE IF NOT EXISTS comments (
  id BIGSERIAL PRIMARY KEY,
  post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  author VARCHAR(40) NOT NULL,
  role VARCHAR(20) NOT NULL CHECK (role IN ('student', 'parent', 'teacher', 'counselor')),
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS subject_insights (
  id BIGSERIAL PRIMARY KEY,
  combination VARCHAR(40) NOT NULL UNIQUE,
  trend VARCHAR(40) NOT NULL,
  heat INTEGER NOT NULL CHECK (heat BETWEEN 0 AND 100),
  match_rate NUMERIC(5, 2) NOT NULL,
  advice TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_track ON posts(track) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_category ON posts(category) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_posts_electives ON posts USING GIN(electives);
CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id) WHERE deleted_at IS NULL;

INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, likes_count, comments_count, favorites_count, created_at)
VALUES
  ('小周同学', 'student', '物化生适合目标不太明确的人吗？', '我现在数学和物理还可以，化学中上，生物背诵压力能接受。想听听大家对物化生后续专业覆盖和学习强度的真实感受。', 'physics', ARRAY['chemistry','biology'], 'question', '高一', '浙江', 128, 3, 42, now() - interval '2 hours'),
  ('林妈妈', 'parent', '孩子想选史政地，家长应该怎么判断风险？', '孩子文科表达不错，但我们担心专业选择变窄。想请教史政地在赋分和未来专业方向上要提前注意什么。', 'history', ARRAY['politics','geography'], 'question', '高一', '山东', 96, 2, 31, now() - interval '5 hours'),
  ('陈老师', 'teacher', '从最近三届学生看物化地的优劣势', '物化地通常适合物理基础稳、空间理解强、但不想承受生物记忆量的学生。它的优势是工科覆盖较好，地理赋分在部分地区也较友好。', 'physics', ARRAY['chemistry','geography'], 'experience', '高一', '广东', 212, 4, 88, now() - interval '1 day'),
  ('升学规划师A', 'counselor', '选科不应只看热门，还要看三类匹配', '建议把学科能力、目标专业、所在省份赋分规则放在同一张表里评估。热门组合不一定适合每个学生。', 'physics', ARRAY['biology','geography'], 'data', '高一', '江苏', 184, 1, 76, now() - interval '2 days'),
  ('选科研究所', 'counselor', '2026届各组合专业覆盖率汇总', '基于多省教育考试院公开数据整理。选科前一定要把目标专业组、学校层次、赋分规则一起看，不要只看一个覆盖率数字。', 'physics', ARRAY['chemistry','biology'], 'data', '高一', '全国', 1100, 2, 480, now() - interval '3 days'),
  ('奋斗的高三党', 'student', '从学渣到班级前十：我的逆袭经验', '选科后成绩波动很大，找到适合自己的节奏最重要。物理重建模型，化学整理反应链，生物用错题回扣课本。', 'physics', ARRAY['chemistry','biology'], 'experience', '高三', '湖北', 611, 1, 199, now() - interval '4 days')
ON CONFLICT DO NOTHING;

INSERT INTO comments (post_id, author, role, content, created_at)
VALUES
  (1, '一只铅笔', 'student', '我也是物化生，最大感受是节奏很满，但专业覆盖确实安心。', now() - interval '90 minutes'),
  (1, '王老师', 'teacher', '可以先看校内排名稳定性，不要只看一次月考。', now() - interval '60 minutes'),
  (1, '海淀家长', 'parent', '我们家最后用三次考试均分和兴趣访谈一起判断。', now() - interval '30 minutes'),
  (2, '南方同学', 'student', '史政地不是轻松组合，政治和地理都需要持续整理。', now() - interval '4 hours'),
  (2, '周顾问', 'counselor', '建议先列出不能报考的专业清单，家长和孩子一起确认能否接受。', now() - interval '3 hours'),
  (3, '小许', 'student', '物化地的地理确实更像半理科，图表题很多。', now() - interval '20 hours'),
  (3, '李爸爸', 'parent', '这个组合对目标工科的孩子比较友好。', now() - interval '18 hours'),
  (3, '陈老师', 'teacher', '也要留意本省高校专业组选科要求，每年都要看最新版本。', now() - interval '16 hours'),
  (3, '高一新生', 'student', '感谢分享，终于看到不是只讲覆盖率的经验。', now() - interval '14 hours'),
  (4, '小叶', 'student', '三类匹配这个方法很清楚，我准备做一张表。', now() - interval '1 day')
ON CONFLICT DO NOTHING;

INSERT INTO subject_insights (combination, trend, heat, match_rate, advice)
VALUES
  ('物理 + 化学 + 生物', '专业覆盖高，学习强度高', 96, 91.50, '适合数理基础稳定、能承受连续刷题和记忆任务的学生。'),
  ('物理 + 化学 + 地理', '工科友好，地理赋分需看省份', 88, 84.20, '适合物理化学较稳且喜欢图表、空间分析的学生。'),
  ('物理 + 生物 + 地理', '压力相对均衡，专业覆盖中高', 74, 78.40, '适合想保留理工方向但化学压力较大的学生。'),
  ('历史 + 政治 + 地理', '人文社科清晰，专业边界明确', 81, 73.10, '适合表达、记忆、材料分析能力强且接受专业范围收窄的学生。'),
  ('历史 + 化学 + 生物', '跨向组合，小众但有医学相关想象', 43, 62.30, '需要重点核对目标院校专业组选科要求，避免只凭兴趣判断。')
ON CONFLICT (combination) DO UPDATE
SET trend = EXCLUDED.trend,
    heat = EXCLUDED.heat,
    match_rate = EXCLUDED.match_rate,
    advice = EXCLUDED.advice,
    updated_at = now();
