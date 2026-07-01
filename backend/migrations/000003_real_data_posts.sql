INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, likes_count, comments_count, favorites_count, created_at)
SELECT '数据观察员', 'counselor', '浙江2024选考目录：物化双选和不限选考几乎持平',
       '基于浙江省教育考试院目录的公开统计，本科层次 34054 个院校专业（类）中，不限选考 15201 个，占 44.6%；要求物理、化学均须选考 15198 个，也约占 44.6%。中肯建议：物化确实保留大量理工农医路径，但对物理、化学连续投入要求高，不适合只为了“覆盖率”盲选。来源：浙江省教育考试院目录及自主选拔在线汇总。',
       'physics', ARRAY['chemistry','biology'], 'data', '高一', '浙江', 433, 2, 260, now() - interval '35 minutes'
WHERE NOT EXISTS (SELECT 1 FROM posts WHERE title = '浙江2024选考目录：物化双选和不限选考几乎持平');

INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, likes_count, comments_count, favorites_count, created_at)
SELECT '升学规划师B', 'counselor', '上海2024：为什么物化双选专业组明显变多？',
       '解放日报公开报道统计，上海新版 2024 选考要求共有 21229 个专业，其中不限选科 9895 个，物化双选 9124 个，占比约 42.98%。中肯建议：上海考生不要只问“这个专业要不要物理”，还要看同一院校专业组里是否同时要求化学。',
       'physics', ARRAY['chemistry','geography'], 'data', '高一', '上海', 366, 1, 188, now() - interval '70 minutes'
WHERE NOT EXISTS (SELECT 1 FROM posts WHERE title = '上海2024：为什么物化双选专业组明显变多？');

INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, likes_count, comments_count, favorites_count, created_at)
SELECT '甘肃高考观察', 'counselor', '甘肃2024选科要求：含物理专业过半，物化双选约45.98%',
       '阳光高考发布的甘肃选考要求显示，本科专业 34670 条，不提科目要求占 43.76%，含物理占 51.79%，含化学占 46.46%，含“物理+化学”占 45.98%。中肯建议：物理方向学生不要只问“覆盖率”，还要分清目标专业是否同时卡化学。',
       'physics', ARRAY['chemistry','biology'], 'data', '高一', '甘肃', 318, 1, 166, now() - interval '95 minutes'
WHERE NOT EXISTS (SELECT 1 FROM posts WHERE title = '甘肃2024选科要求：含物理专业过半，物化双选约45.98%');

INSERT INTO comments (post_id, author, role, content, created_at)
SELECT p.id, '王老师', 'teacher', '这类数据最适合拿来做排除法：先看目标专业硬性要求，再看孩子能否长期稳定投入。', now() - interval '20 minutes'
FROM posts p
WHERE p.title IN (
  '浙江2024选考目录：物化双选和不限选考几乎持平',
  '上海2024：为什么物化双选专业组明显变多？',
  '甘肃2024选科要求：含物理专业过半，物化双选约45.98%'
)
AND NOT EXISTS (
  SELECT 1 FROM comments c WHERE c.post_id = p.id AND c.author = '王老师' AND c.content LIKE '这类数据最适合拿来做排除法%'
);
