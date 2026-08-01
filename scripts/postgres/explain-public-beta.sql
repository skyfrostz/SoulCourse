EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, title, created_at
FROM posts
WHERE deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT 20;

EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, title
FROM posts
WHERE deleted_at IS NULL
  AND search_vector @@ plainto_tsquery('simple', '选科')
ORDER BY created_at DESC, id DESC
LIMIT 20;

EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, type, created_at
FROM notifications
WHERE recipient_user_id = 1
  AND (created_at < now() OR (created_at = now() AND id < 9223372036854775807))
ORDER BY created_at DESC, id DESC
LIMIT 30;

EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, content, created_at
FROM direct_messages
WHERE ((sender_user_id = 1 AND recipient_user_id = 2)
    OR (sender_user_id = 2 AND recipient_user_id = 1))
ORDER BY created_at DESC, id DESC
LIMIT 50;
