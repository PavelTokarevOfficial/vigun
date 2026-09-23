ALTER TABLE instagram_accounts
  ADD COLUMN token_updated_at TIMESTAMPTZ,
  ADD COLUMN token_expires_at TIMESTAMPTZ,
  ADD COLUMN token_last_checked_at TIMESTAMPTZ,
  ADD COLUMN verified_username TEXT,
  ADD COLUMN account_type TEXT;

UPDATE instagram_accounts
SET token_updated_at=updated_at;

ALTER TABLE instagram_accounts
  ALTER COLUMN token_updated_at SET DEFAULT now(),
  ALTER COLUMN token_updated_at SET NOT NULL;
