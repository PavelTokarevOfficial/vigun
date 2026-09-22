UPDATE instagram_accounts
SET token_updated_at=updated_at
WHERE token_expires_at IS NULL
  AND token_last_checked_at IS NULL;
