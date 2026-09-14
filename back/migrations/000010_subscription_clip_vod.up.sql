ALTER TABLE subscription_clips
    ADD COLUMN twitch_video_id TEXT,
    ADD COLUMN vod_offset INTEGER;
