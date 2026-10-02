package media

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Library removes source media or an entire clip that is not currently being processed.
// Completed renders survive source deletion and remain available independently.
type Library struct {
	db      *pgxpool.Pool
	storage Storage
}

func NewLibrary(db *pgxpool.Pool, storage Storage) *Library {
	return &Library{db: db, storage: storage}
}

func (l *Library) SourceURL(ctx context.Context, clipID string) (string, error) {
	key, err := l.sourceKey(ctx, clipID)
	if err != nil {
		return "", err
	}
	return l.storage.PresignGet(ctx, key, 15*time.Minute)
}

func (l *Library) Source(ctx context.Context, clipID string) (Object, error) {
	key, err := l.sourceKey(ctx, clipID)
	if err != nil {
		return Object{}, err
	}
	return l.storage.Get(ctx, key)
}

func (l *Library) sourceKey(ctx context.Context, clipID string) (string, error) {
	var key string
	if err := l.db.QueryRow(ctx, `SELECT COALESCE(m.storage_key,v.storage_key,a.storage_key) FROM clips c
		LEFT JOIN LATERAL (SELECT storage_key FROM media_files WHERE clip_id=c.id AND type='source' ORDER BY created_at DESC LIMIT 1) m ON true
		LEFT JOIN source_videos v ON v.id=c.source_video_id
		LEFT JOIN assets a ON a.id=c.source_asset_id
		WHERE c.id=$1 AND COALESCE(m.storage_key,v.storage_key,a.storage_key) IS NOT NULL`, clipID).Scan(&key); err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("downloaded source not found")
		}
		return "", err
	}
	return key, nil
}

func (l *Library) DeleteSourceOrClip(ctx context.Context, clipID string) error {
	tx, err := l.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	if err = tx.QueryRow(ctx, "SELECT status::text FROM clips WHERE id=$1 FOR UPDATE", clipID).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("clip not found")
		}
		return err
	}
	if status != "saved" && status != "downloaded" && status != "failed" && status != "completed" {
		return fmt.Errorf("only favorite, downloaded, failed, or completed clips can be deleted")
	}
	var hasActiveJob bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM processing_jobs
		WHERE (clip_id=$1 OR clip_id IN (SELECT id FROM clips WHERE source_clip_id=$1))
		AND status IN ('pending','running')
	)`, clipID).Scan(&hasActiveJob); err != nil {
		return err
	}
	if hasActiveJob {
		return fmt.Errorf("clip has an active job and cannot be deleted")
	}
	var hasRender bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM media_files WHERE clip_id=$1 AND type='render'
	)`, clipID).Scan(&hasRender); err != nil {
		return err
	}

	mediaQuery := "SELECT storage_key FROM media_files WHERE clip_id=$1"
	if hasRender {
		mediaQuery += " AND type<>'render'"
	}
	rows, err := tx.Query(ctx, mediaQuery, clipID)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, key)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, key := range keys {
		if err = l.storage.Delete(ctx, key); err != nil {
			return fmt.Errorf("delete %q from object storage: %w", key, err)
		}
	}
	if hasRender {
		if _, err = tx.Exec(ctx, "DELETE FROM media_files WHERE clip_id=$1 AND type<>'render'", clipID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "UPDATE clips SET status='completed',error=NULL,updated_at=now() WHERE id=$1", clipID); err != nil {
			return err
		}
	} else {
		if _, err = tx.Exec(ctx, "DELETE FROM clips WHERE id=$1", clipID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
