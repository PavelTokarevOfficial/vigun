ALTER TABLE processing_jobs
  DROP COLUMN IF EXISTS cancel_requested;

DROP INDEX IF EXISTS clips_source_clip_idx;
DROP INDEX IF EXISTS clips_origin_asset_idx;

ALTER TABLE clips
  DROP COLUMN IF EXISTS source_clip_id,
  DROP COLUMN IF EXISTS origin_asset_id;

-- PostgreSQL enum values are intentionally retained on rollback because
-- removing enum values is unsafe while historical rows may reference them.
