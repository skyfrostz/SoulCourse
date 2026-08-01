\set ON_ERROR_STOP on
\if :{?target_year}
\else
  \echo 'target_year is required'
  \quit 2
\endif

DELETE FROM requirements;
DELETE FROM policies;
DELETE FROM sources;
DELETE FROM provinces WHERE code = 'GD';

WITH province AS (
  INSERT INTO provinces (code, name, region, coverage_status, data_year, records_count, captured_at, methodology)
  VALUES ('GD', '广东', '华南', 'verified', :'target_year', 2, now(), 'CI fixture for validating the release gate')
  RETURNING id
), source AS (
  INSERT INTO sources (province_id, name, url, asset_key, file_hash, data_year, scope, captured_at, methodology, coverage_status)
  SELECT id, 'CI Guangdong source', 'https://example.invalid/ci-source.pdf', 'ci/gd/source.pdf',
    repeat('a', 64), :'target_year', '广东', now(), 'CI fixture only', 'verified'
  FROM province
  RETURNING id, province_id
), policy AS (
  INSERT INTO policies (province_id, source_id, title, type, scope, coverage_status, data_year, captured_at, summary, methodology, tags, url)
  SELECT province_id, id, 'CI policy', '政策', '广东', 'verified', :'target_year', now(), '', 'CI fixture only', '[]', 'https://example.invalid/ci-policy'
  FROM source
)
INSERT INTO requirements (province_id, source_id, title, major_code, type, scope, required_subjects, coverage_status, data_year, captured_at, summary, methodology, tags, url)
SELECT province_id, id, 'CI requirement', 'CI0001', '专业要求', '广东', '["physics"]', 'verified', :'target_year', now(), '', 'CI fixture only', '[]', 'https://example.invalid/ci-requirement'
FROM source;
