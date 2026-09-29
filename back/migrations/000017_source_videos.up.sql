CREATE TABLE source_videos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL CHECK (length(trim(name)) > 0),
  mime_type TEXT NOT NULL,
  size BIGINT NOT NULL CHECK (size > 0),
  duration DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (duration >= 0),
  storage_key TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE clips
  ADD COLUMN source_video_id UUID REFERENCES source_videos(id) ON DELETE RESTRICT;

CREATE INDEX clips_source_video_idx ON clips(source_video_id)
  WHERE source_video_id IS NOT NULL;

-- Library fragments still use the established clip/render pipeline. A dedicated
-- system author keeps the existing non-null streamer contract intact.
INSERT INTO streamers(id, twitch_login, display_name, twitch_user_id)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  '__video_library__',
  'Видео / Сериалы',
  NULL
)
ON CONFLICT (id) DO NOTHING;
