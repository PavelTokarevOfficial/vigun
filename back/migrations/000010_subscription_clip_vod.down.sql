ALTER TABLE subscription_clips
    DROP COLUMN IF EXISTS vod_offset,
    DROP COLUMN IF EXISTS twitch_video_id;
