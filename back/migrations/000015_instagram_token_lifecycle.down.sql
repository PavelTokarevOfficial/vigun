ALTER TABLE instagram_accounts
  DROP COLUMN IF EXISTS account_type,
  DROP COLUMN IF EXISTS verified_username,
  DROP COLUMN IF EXISTS token_last_checked_at,
  DROP COLUMN IF EXISTS token_expires_at,
  DROP COLUMN IF EXISTS token_updated_at;
