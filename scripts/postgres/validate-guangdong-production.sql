\set ON_ERROR_STOP on
\if :{?target_year}
\else
  \echo 'target_year is required (for example: -v target_year=2026)'
  \quit 2
\endif

BEGIN READ ONLY;
SELECT set_config('soulcourse.target_year', :'target_year', true);

DO $$
DECLARE
  gd_id BIGINT;
  invalid_count INTEGER;
  expected_year INTEGER := current_setting('soulcourse.target_year')::INTEGER;
BEGIN
  SELECT id INTO gd_id FROM provinces WHERE code = 'GD' AND name = '广东';
  IF gd_id IS NULL THEN
    RAISE EXCEPTION 'missing Guangdong province row (code=GD, name=广东)';
  END IF;

  IF EXISTS (
    SELECT 1 FROM provinces WHERE id = gd_id AND (
      coverage_status <> 'verified' OR data_year <> expected_year OR records_count <= 0
      OR captured_at > now() OR btrim(methodology) = ''
    )
  ) THEN
    RAISE EXCEPTION 'Guangdong province metadata is incomplete or unverified';
  END IF;

  SELECT count(*) INTO invalid_count FROM sources WHERE province_id = gd_id AND (
    coverage_status <> 'verified' OR btrim(name) = '' OR url !~ '^https://'
    OR btrim(coalesce(asset_key, '')) = '' OR file_hash !~ '^[0-9a-fA-F]{64}$'
    OR data_year <> expected_year OR btrim(scope) = '' OR captured_at > now()
    OR btrim(methodology) = ''
  );
  IF invalid_count > 0 OR NOT EXISTS (SELECT 1 FROM sources WHERE province_id = gd_id) THEN
    RAISE EXCEPTION 'Guangdong sources missing or invalid: % invalid rows', invalid_count;
  END IF;

  SELECT count(*) INTO invalid_count
  FROM policies p JOIN sources s ON s.id = p.source_id
  WHERE p.province_id = gd_id AND p.deleted_at IS NULL AND (
    p.coverage_status <> 'verified' OR s.province_id IS DISTINCT FROM gd_id
    OR s.coverage_status <> 'verified' OR btrim(p.title) = '' OR btrim(p.type) = ''
    OR btrim(p.scope) = '' OR p.data_year <> expected_year OR p.captured_at > now()
    OR btrim(p.methodology) = '' OR jsonb_typeof(p.tags) <> 'array'
  );
  IF invalid_count > 0 OR NOT EXISTS (SELECT 1 FROM policies WHERE province_id = gd_id AND deleted_at IS NULL) THEN
    RAISE EXCEPTION 'Guangdong policies missing or invalid: % invalid rows', invalid_count;
  END IF;

  SELECT count(*) INTO invalid_count
  FROM requirements r JOIN sources s ON s.id = r.source_id
  WHERE r.province_id = gd_id AND r.deleted_at IS NULL AND (
    r.coverage_status <> 'verified' OR s.province_id IS DISTINCT FROM gd_id
    OR s.coverage_status <> 'verified' OR btrim(r.title) = '' OR btrim(r.major_code) = ''
    OR btrim(r.type) = '' OR btrim(r.scope) = '' OR r.data_year <> expected_year
    OR r.captured_at > now() OR btrim(r.methodology) = ''
    OR jsonb_typeof(r.required_subjects) <> 'array' OR jsonb_array_length(r.required_subjects) = 0
    OR jsonb_typeof(r.tags) <> 'array'
  );
  IF invalid_count > 0 OR NOT EXISTS (SELECT 1 FROM requirements WHERE province_id = gd_id AND deleted_at IS NULL) THEN
    RAISE EXCEPTION 'Guangdong requirements missing or invalid: % invalid rows', invalid_count;
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_namespace n ON n.oid = c.connamespace
    WHERE n.nspname = 'public' AND c.contype = 'f' AND NOT c.convalidated
  ) THEN
    RAISE EXCEPTION 'database contains unvalidated foreign keys';
  END IF;

  IF EXISTS (
    SELECT 1 FROM provinces p WHERE p.id = gd_id AND p.records_count <>
      (SELECT count(*) FROM policies WHERE province_id = gd_id AND deleted_at IS NULL) +
      (SELECT count(*) FROM requirements WHERE province_id = gd_id AND deleted_at IS NULL)
  ) THEN
    RAISE EXCEPTION 'Guangdong records_count does not match published records';
  END IF;
END $$;

SELECT
  p.data_year,
  p.records_count,
  (SELECT count(*) FROM sources WHERE province_id = p.id) AS sources,
  (SELECT count(*) FROM policies WHERE province_id = p.id AND deleted_at IS NULL) AS policies,
  (SELECT count(*) FROM requirements WHERE province_id = p.id AND deleted_at IS NULL) AS requirements
FROM provinces p WHERE p.code = 'GD';

ROLLBACK;
