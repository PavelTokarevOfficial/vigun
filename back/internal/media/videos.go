package media

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Video struct {
	ID              string    `json:"id"`
	ClipID          string    `json:"clipId"`
	ProcessingJobID string    `json:"processingJobId"`
	Title           string    `json:"title"`
	Streamer        string    `json:"streamer"`
	TwitchURL       string    `json:"twitchUrl"`
	ThumbnailURL    string    `json:"thumbnailUrl"`
	TemplateName    string    `json:"templateName"`
	StorageKey      string    `json:"storageKey"`
	URL             string    `json:"url"`
	CreatedAt       time.Time `json:"createdAt"`
}
type Videos struct {
	db      *pgxpool.Pool
	storage Storage
}

func NewVideos(db *pgxpool.Pool, s Storage) *Videos { return &Videos{db, s} }
func (v *Videos) List(ctx context.Context) ([]Video, error) {
	rows, e := v.db.Query(ctx, `
		SELECT m.id,c.id,COALESCE(m.processing_job_id::text,''),c.title,s.display_name,c.twitch_url,COALESCE(c.thumbnail_url,''),
			COALESCE(j.template_snapshot->>'templateName',''),m.storage_key,m.created_at
		FROM media_files m
		JOIN clips c ON c.id=m.clip_id
		JOIN streamers s ON s.id=c.streamer_id
		LEFT JOIN processing_jobs j ON j.id=m.processing_job_id
		WHERE m.type='render'
		ORDER BY m.created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		var x Video
		if e = rows.Scan(&x.ID, &x.ClipID, &x.ProcessingJobID, &x.Title, &x.Streamer, &x.TwitchURL, &x.ThumbnailURL, &x.TemplateName, &x.StorageKey, &x.CreatedAt); e != nil {
			return nil, e
		}
		x.URL, e = v.storage.PresignGet(ctx, x.StorageKey, 15*time.Minute)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// Download opens one rendered result through the storage boundary.
func (v *Videos) Download(ctx context.Context, id string) (Object, string, error) {
	var title, key string
	if err := v.db.QueryRow(ctx, `
		SELECT c.title,m.storage_key
		FROM media_files m
		JOIN clips c ON c.id=m.clip_id
		WHERE m.id=$1 AND m.type='render'`, id).Scan(&title, &key); err != nil {
		if err == pgx.ErrNoRows {
			return Object{}, "", fmt.Errorf("rendered video not found")
		}
		return Object{}, "", err
	}
	object, err := v.storage.Get(ctx, key)
	if err != nil {
		return Object{}, "", fmt.Errorf("open rendered video from object storage: %w", err)
	}
	return object, renderFilename(title), nil
}

func (v *Videos) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := v.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM media_files WHERE id=$1 AND type='render')`, id).Scan(&exists)
	return exists, err
}

func renderFilename(title string) string {
	title = strings.TrimSpace(strings.Map(func(char rune) rune {
		if char < 32 || char == '/' || char == '\\' {
			return '-'
		}
		return char
	}, title))
	if title == "" {
		title = "video"
	}
	return title + ".mp4"
}

// Delete removes one rendered result while preserving the clip and its source.
func (v *Videos) Delete(ctx context.Context, id string) error {
	tx, err := v.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var clipID, key string
	if err = tx.QueryRow(ctx, `SELECT clip_id,storage_key FROM media_files WHERE id=$1 AND type='render' FOR UPDATE`, id).Scan(&clipID, &key); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("rendered video not found")
		}
		return err
	}
	if err = v.storage.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete rendered video from object storage: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM media_files WHERE id=$1`, id); err != nil {
		return err
	}
	var hasRender, hasSource bool
	if err = tx.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM media_files WHERE clip_id=$1 AND type='render'),
		EXISTS(SELECT 1 FROM media_files WHERE clip_id=$1 AND type='source')`, clipID).Scan(&hasRender, &hasSource); err != nil {
		return err
	}
	if !hasRender && !hasSource {
		if _, err = tx.Exec(ctx, "DELETE FROM clips WHERE id=$1", clipID); err != nil {
			return err
		}
	} else {
		status := "downloaded"
		if hasRender {
			status = "completed"
		}
		if _, err = tx.Exec(ctx, "UPDATE clips SET status=$2,error=NULL,updated_at=now() WHERE id=$1", clipID, status); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
