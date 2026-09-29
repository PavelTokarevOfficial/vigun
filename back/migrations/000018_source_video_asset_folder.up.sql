CREATE TABLE source_video_settings (
  singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK (singleton),
  folder_id UUID REFERENCES asset_folders(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO source_video_settings(singleton) VALUES (true);
ALTER TABLE clips ADD COLUMN source_asset_id UUID REFERENCES assets(id) ON DELETE RESTRICT;
CREATE INDEX clips_source_asset_idx ON clips(source_asset_id) WHERE source_asset_id IS NOT NULL;
