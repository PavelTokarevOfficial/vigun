DROP TABLE IF EXISTS subscription_clips;

ALTER TABLE streamers
  DROP COLUMN IF EXISTS subscription_synced_at,
  DROP COLUMN IF EXISTS subscribed;
