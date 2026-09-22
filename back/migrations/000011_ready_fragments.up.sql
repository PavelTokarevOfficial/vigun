ALTER TABLE clips
  ADD COLUMN is_ready_fragment BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN edit_timeline JSONB;

CREATE INDEX clips_ready_fragment_idx
  ON clips(is_ready_fragment, created_at DESC)
  WHERE is_ready_fragment = true;
