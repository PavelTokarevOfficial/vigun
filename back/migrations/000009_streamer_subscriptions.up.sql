ALTER TABLE streamers
  ADD COLUMN subscribed BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN subscription_synced_at TIMESTAMPTZ;

CREATE TABLE subscription_clips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  streamer_id UUID NOT NULL REFERENCES streamers(id) ON DELETE CASCADE,
  twitch_clip_id TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  twitch_url TEXT NOT NULL,
  thumbnail_url TEXT,
  duration DOUBLE PRECISION,
  twitch_created_at TIMESTAMPTZ NOT NULL,
  viewed_at TIMESTAMPTZ,
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX subscription_clips_streamer_created_idx
  ON subscription_clips(streamer_id, twitch_created_at DESC);
