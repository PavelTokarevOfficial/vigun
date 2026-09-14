package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/subscription"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) List(ctx context.Context) ([]subscription.Feed, error) {
	rows, err := r.db.Query(ctx, `SELECT id,twitch_login,display_name,subscription_synced_at
		FROM streamers WHERE subscribed=true ORDER BY priority DESC,created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeds := make([]subscription.Feed, 0)
	feedIndex := make(map[string]int)
	for rows.Next() {
		var feed subscription.Feed
		if err = rows.Scan(&feed.StreamerID, &feed.TwitchLogin, &feed.DisplayName, &feed.LastSyncedAt); err != nil {
			return nil, err
		}
		feed.Clips = []subscription.Clip{}
		feedIndex[feed.StreamerID] = len(feeds)
		feeds = append(feeds, feed)
	}
	if err = rows.Err(); err != nil || len(feeds) == 0 {
		return feeds, err
	}

	clipRows, err := r.db.Query(ctx, `SELECT sc.streamer_id,sc.twitch_clip_id,sc.title,sc.twitch_url,
		COALESCE(sc.thumbnail_url,''),COALESCE(sc.duration,0),sc.twitch_created_at,
		EXISTS(SELECT 1 FROM clips c WHERE c.twitch_clip_id=sc.twitch_clip_id),sc.viewed_at IS NOT NULL
		FROM subscription_clips sc
		JOIN streamers st ON st.id=sc.streamer_id
		WHERE st.subscribed=true AND sc.twitch_created_at >= now() - interval '7 days'
		ORDER BY sc.twitch_created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer clipRows.Close()

	for clipRows.Next() {
		var streamerID string
		var clip subscription.Clip
		if err = clipRows.Scan(&streamerID, &clip.ID, &clip.Title, &clip.URL, &clip.ThumbnailURL, &clip.Duration, &clip.CreatedAt, &clip.Saved, &clip.Viewed); err != nil {
			return nil, err
		}
		if index, ok := feedIndex[streamerID]; ok {
			feeds[index].Clips = append(feeds[index].Clips, clip)
		}
	}
	return feeds, clipRows.Err()
}

func (r *SubscriptionRepository) Target(ctx context.Context, streamerID string) (subscription.Target, error) {
	target := subscription.Target{StreamerID: streamerID}
	err := r.db.QueryRow(ctx, `SELECT COALESCE(twitch_user_id,''),twitch_login
		FROM streamers WHERE id=$1 AND subscribed=true`, streamerID).Scan(&target.TwitchID, &target.TwitchLogin)
	if err == pgx.ErrNoRows {
		return subscription.Target{}, fmt.Errorf("streamer is not subscribed")
	}
	return target, err
}

func (r *SubscriptionRepository) UpdateIdentity(ctx context.Context, streamerID string, user subscription.RemoteUser) error {
	_, err := r.db.Exec(ctx, `UPDATE streamers SET twitch_user_id=$2,display_name=$3,updated_at=now()
		WHERE id=$1`, streamerID, user.ID, user.DisplayName)
	return err
}

func (r *SubscriptionRepository) SaveWindow(ctx context.Context, streamerID string, clips []subscription.RemoteClip, startedAt, syncedAt time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, clip := range clips {
		_, err = tx.Exec(ctx, `INSERT INTO subscription_clips(
			streamer_id,twitch_clip_id,title,twitch_url,thumbnail_url,duration,twitch_created_at,synced_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(twitch_clip_id) DO UPDATE SET
			streamer_id=EXCLUDED.streamer_id,title=EXCLUDED.title,twitch_url=EXCLUDED.twitch_url,
			thumbnail_url=EXCLUDED.thumbnail_url,duration=EXCLUDED.duration,
			twitch_created_at=EXCLUDED.twitch_created_at,synced_at=EXCLUDED.synced_at`,
			streamerID, clip.ID, clip.Title, clip.URL, clip.ThumbnailURL, clip.Duration, clip.CreatedAt, syncedAt)
		if err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM subscription_clips
		WHERE streamer_id=$1 AND twitch_created_at < $2`, streamerID, startedAt); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE streamers SET subscription_synced_at=$2,updated_at=now() WHERE id=$1`, streamerID, syncedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *SubscriptionRepository) MarkViewed(ctx context.Context, clipID string) error {
	result, err := r.db.Exec(ctx, `UPDATE subscription_clips SET viewed_at=COALESCE(viewed_at,now())
		WHERE twitch_clip_id=$1`, clipID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription clip not found")
	}
	return nil
}
