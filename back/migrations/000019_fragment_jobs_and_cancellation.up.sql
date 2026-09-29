ALTER TYPE job_type ADD VALUE IF NOT EXISTS 'fragment';
ALTER TYPE job_status ADD VALUE IF NOT EXISTS 'canceled';

ALTER TABLE clips
  ADD COLUMN source_clip_id UUID REFERENCES clips(id) ON DELETE SET NULL,
  ADD COLUMN origin_asset_id UUID REFERENCES assets(id) ON DELETE SET NULL;

CREATE INDEX clips_source_clip_idx
  ON clips(source_clip_id)
  WHERE source_clip_id IS NOT NULL;

CREATE INDEX clips_origin_asset_idx
  ON clips(origin_asset_id)
  WHERE origin_asset_id IS NOT NULL;

ALTER TABLE processing_jobs
  ADD COLUMN cancel_requested BOOLEAN NOT NULL DEFAULT false;
