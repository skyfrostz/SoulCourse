CREATE TABLE IF NOT EXISTS email_verification_codes (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(120) NOT NULL,
  code_hash CHAR(64) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_email_verification_codes_email_created
  ON email_verification_codes (lower(email), created_at DESC);

CREATE INDEX IF NOT EXISTS idx_email_verification_codes_active
  ON email_verification_codes (lower(email), expires_at DESC)
  WHERE used_at IS NULL;
