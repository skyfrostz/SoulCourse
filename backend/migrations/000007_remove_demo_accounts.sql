DELETE FROM admin_content_records
WHERE id IN ('user-demo-student', 'user-demo-parent')
   OR summary IN ('demo@student.local', 'demo@parent.local');

UPDATE users
SET deleted_at = now()
WHERE lower(email) IN ('demo@student.local', 'demo@parent.local')
  AND deleted_at IS NULL;
