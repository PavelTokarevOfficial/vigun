DROP INDEX IF EXISTS clips_ready_fragment_idx;

ALTER TABLE clips
  DROP COLUMN IF EXISTS edit_timeline,
  DROP COLUMN IF EXISTS is_ready_fragment;
