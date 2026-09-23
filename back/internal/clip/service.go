package clip

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/finde-clip/finde-v2/back/infrastructure/twitch"
	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/videotemplate"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Service struct {
	db        *pgxpool.Pool
	twitch    *twitch.Client
	templates *videotemplate.Service
}
type Local struct {
	ID              string                `json:"id"`
	StreamerID      string                `json:"streamerId"`
	StreamerName    string                `json:"streamerName"`
	Title           string                `json:"title"`
	TwitchClipID    string                `json:"twitchClipId"`
	ThumbnailURL    string                `json:"thumbnailUrl"`
	Duration        float64               `json:"duration"`
	HasSource       bool                  `json:"hasSource"`
	Status          string                `json:"status"`
	Error           string                `json:"error"`
	CurrentStep     string                `json:"currentStep"`
	Progress        int                   `json:"progress"`
	LastJobType     string                `json:"lastJobType"`
	LastJobStatus   string                `json:"lastJobStatus"`
	IsReadyFragment bool                  `json:"isReadyFragment"`
	EditTimeline    *composition.Timeline `json:"editTimeline,omitempty"`
}

func (s *Service) List(ctx context.Context) ([]Local, error) {
	rows, e := s.db.Query(ctx, `SELECT c.id,c.streamer_id,s.display_name,c.title,c.twitch_clip_id,COALESCE(c.thumbnail_url,''),COALESCE(c.duration,0),
		EXISTS(SELECT 1 FROM media_files m WHERE m.clip_id=c.id AND m.type='source'),c.status,COALESCE(c.error,''),
		COALESCE(j.current_step,''),COALESCE(j.progress,0),COALESCE(j.type::text,''),COALESCE(j.status::text,''),
		c.is_ready_fragment,c.edit_timeline
		FROM clips c JOIN streamers s ON s.id=c.streamer_id
		LEFT JOIN LATERAL (SELECT current_step,progress,type,status FROM processing_jobs WHERE clip_id=c.id ORDER BY created_at DESC LIMIT 1) j ON true
		ORDER BY c.created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Local{}
	for rows.Next() {
		var x Local
		if e = rows.Scan(&x.ID, &x.StreamerID, &x.StreamerName, &x.Title, &x.TwitchClipID, &x.ThumbnailURL, &x.Duration, &x.HasSource, &x.Status, &x.Error, &x.CurrentStep, &x.Progress, &x.LastJobType, &x.LastJobStatus, &x.IsReadyFragment, &x.EditTimeline); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Service) EnqueueDownload(ctx context.Context, id string) error {
	var jobID string
	e := s.db.QueryRow(ctx, `WITH queued AS (
		INSERT INTO processing_jobs(clip_id,type)
		SELECT c.id,'download' FROM clips c
		WHERE c.id=$1 AND c.status='saved'
		AND NOT EXISTS (SELECT 1 FROM processing_jobs j WHERE j.clip_id=c.id AND j.type='download' AND j.status IN ('pending','running'))
		ON CONFLICT DO NOTHING
		RETURNING clip_id
	)
	UPDATE clips c SET status='downloading',error=NULL,updated_at=now()
	FROM queued WHERE c.id=queued.clip_id
	RETURNING c.id`, id).Scan(&jobID)
	if e == pgx.ErrNoRows {
		return fmt.Errorf("clip is not in favorites or already has an active download")
	}
	return e
}

func (s *Service) EnqueueProcess(ctx context.Context, id, templateID string, draft *composition.Config) error {
	var snapshot []byte
	var e error
	if draft == nil {
		snapshot, e = s.templates.Snapshot(ctx, templateID)
	} else {
		config := *draft
		config.Layers = append([]composition.Layer(nil), draft.Layers...)
		var streamerName string
		if e = s.db.QueryRow(ctx, `SELECT s.display_name FROM clips c JOIN streamers s ON s.id=c.streamer_id WHERE c.id=$1`, id).Scan(&streamerName); e != nil {
			if e == pgx.ErrNoRows {
				return fmt.Errorf("clip not found")
			}
			return e
		}
		for index := range config.Layers {
			if config.Layers[index].Type == "text" && config.Layers[index].TextSource == "streamer_name" {
				config.Layers[index].Text = streamerName
			}
		}
		snapshot, e = s.templates.SnapshotConfig(ctx, templateID, config)
	}
	if e != nil {
		return e
	}
	snapshot, e = s.attachClipSources(ctx, id, snapshot)
	if e != nil {
		return e
	}
	var jobID string
	e = s.db.QueryRow(ctx, `WITH queued AS (
		INSERT INTO processing_jobs(clip_id,type,template_id,template_snapshot)
		SELECT c.id,'process',$2,$3 FROM clips c
		WHERE c.id=$1
		AND EXISTS (SELECT 1 FROM media_files m WHERE m.clip_id=c.id AND m.type='source')
		AND NOT EXISTS (SELECT 1 FROM processing_jobs j WHERE j.clip_id=c.id AND j.type='process' AND j.status IN ('pending','running'))
		ON CONFLICT DO NOTHING RETURNING clip_id
	)
	UPDATE clips c SET status='downloaded',error=NULL,updated_at=now()
	FROM queued WHERE c.id=queued.clip_id RETURNING c.id`, id, templateID, snapshot).Scan(&jobID)
	if e == pgx.ErrNoRows {
		var hasSource, hasActiveJob bool
		if lookupErr := s.db.QueryRow(ctx, `SELECT
			EXISTS(SELECT 1 FROM media_files WHERE clip_id=$1 AND type='source'),
			EXISTS(SELECT 1 FROM processing_jobs WHERE clip_id=$1 AND type='process' AND status IN ('pending','running'))`, id).Scan(&hasSource, &hasActiveJob); lookupErr != nil {
			return lookupErr
		}
		if hasActiveJob {
			return fmt.Errorf("render is already queued or running")
		}
		if !hasSource {
			return fmt.Errorf("downloaded source not found")
		}
		return fmt.Errorf("could not queue render")
	}
	return e
}

func (s *Service) attachClipSources(ctx context.Context, primaryID string, raw []byte) ([]byte, error) {
	var snapshot composition.Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode template snapshot: %w", err)
	}
	if snapshot.Config.Timeline == nil {
		return raw, nil
	}
	seen := map[string]bool{}
	for _, asset := range snapshot.Assets {
		seen[asset.ID] = true
	}
	for index := range snapshot.Config.Timeline.Segments {
		segment := &snapshot.Config.Timeline.Segments[index]
		if segment.ClipID == "" || segment.ClipID == primaryID {
			segment.ClipID = ""
			continue
		}
		assetID := "clip-source:" + segment.ClipID
		if !seen[assetID] {
			var storageKey, mimeType string
			err := s.db.QueryRow(ctx, `SELECT m.storage_key,m.mime_type
				FROM clips c JOIN media_files m ON m.clip_id=c.id AND m.type='source'
				WHERE c.id=$1 AND c.is_ready_fragment=true
				ORDER BY m.created_at DESC LIMIT 1`, segment.ClipID).Scan(&storageKey, &mimeType)
			if err == pgx.ErrNoRows {
				return nil, fmt.Errorf("ready fragment %s has no downloaded source", segment.ClipID)
			}
			if err != nil {
				return nil, err
			}
			snapshot.Assets = append(snapshot.Assets, composition.AssetSnapshot{ID: assetID, StorageKey: storageKey, Kind: "video", MIMEType: mimeType})
			seen[assetID] = true
		}
		segment.Source = "asset"
		segment.AssetID = assetID
		segment.ClipID = ""
	}
	return json.Marshal(snapshot)
}

func (s *Service) UpdateFragment(ctx context.Context, id string, ready bool, timeline *composition.Timeline) error {
	if timeline != nil {
		if len(timeline.Segments) == 0 || len(timeline.Segments) > 100 {
			return fmt.Errorf("fragment timeline must contain from 1 to 100 segments")
		}
		for index, segment := range timeline.Segments {
			if segment.Start < 0 || segment.End <= segment.Start || (segment.SourceDuration > 0 && segment.End > segment.SourceDuration+0.001) {
				return fmt.Errorf("fragment segment %d is invalid", index+1)
			}
		}
	}
	data, err := json.Marshal(timeline)
	if err != nil {
		return err
	}
	command, err := s.db.Exec(ctx, `UPDATE clips SET is_ready_fragment=$2,edit_timeline=$3,updated_at=now()
		WHERE id=$1 AND status IN ('downloaded','completed')
		AND EXISTS(SELECT 1 FROM media_files WHERE clip_id=$1 AND type='source')`, id, ready, data)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return fmt.Errorf("only a downloaded clip can become a ready fragment")
	}
	return nil
}

func (s *Service) Retry(ctx context.Context, id string) error {
	var jobType string
	e := s.db.QueryRow(ctx, `SELECT j.type::text
		FROM clips c JOIN LATERAL (
			SELECT type FROM processing_jobs WHERE clip_id=c.id ORDER BY created_at DESC LIMIT 1
		) j ON true
		WHERE c.id=$1 AND c.status='failed'`, id).Scan(&jobType)
	if e == pgx.ErrNoRows {
		return fmt.Errorf("only failed clips can be retried")
	}
	if e != nil {
		return e
	}

	if jobType == "download" {
		if _, e = s.db.Exec(ctx, "UPDATE clips SET status='saved',error=NULL,updated_at=now() WHERE id=$1", id); e != nil {
			return e
		}
		return s.EnqueueDownload(ctx, id)
	}
	if jobType == "process" {
		var templateID *string
		var snapshot []byte
		if e = s.db.QueryRow(ctx, `SELECT template_id,template_snapshot FROM processing_jobs WHERE clip_id=$1 AND type='process' ORDER BY created_at DESC LIMIT 1`, id).Scan(&templateID, &snapshot); e != nil {
			return e
		}
		if templateID == nil || len(snapshot) == 0 {
			defaultID, lookupErr := s.templates.DefaultID(ctx)
			if lookupErr != nil {
				return lookupErr
			}
			snapshot, e = s.templates.Snapshot(ctx, defaultID)
			if e != nil {
				return e
			}
			templateID = &defaultID
		}
		if _, e = s.db.Exec(ctx, "UPDATE clips SET status='downloaded',error=NULL,updated_at=now() WHERE id=$1", id); e != nil {
			return e
		}
		var queued string
		e = s.db.QueryRow(ctx, `INSERT INTO processing_jobs(clip_id,type,template_id,template_snapshot)
			SELECT id,'process',$2,$3 FROM clips WHERE id=$1 RETURNING id`, id, *templateID, snapshot).Scan(&queued)
		return e
	}
	return fmt.Errorf("unknown job type %q", jobType)
}

func New(db *pgxpool.Pool, t *twitch.Client, templates *videotemplate.Service) *Service {
	return &Service{db: db, twitch: t, templates: templates}
}
func (s *Service) Remote(ctx context.Context, streamerID string, startedAt, endedAt time.Time) ([]twitch.Clip, error) {
	var tid, login string
	if e := s.db.QueryRow(ctx, "SELECT COALESCE(twitch_user_id,''),twitch_login FROM streamers WHERE id=$1", streamerID).Scan(&tid, &login); e != nil {
		return nil, e
	}
	if tid == "" {
		u, e := s.twitch.User(ctx, login)
		if e != nil {
			return nil, e
		}
		tid = u.ID
		_, e = s.db.Exec(ctx, "UPDATE streamers SET twitch_user_id=$2,display_name=$3,updated_at=now() WHERE id=$1", streamerID, u.ID, u.DisplayName)
		if e != nil {
			return nil, e
		}
	}
	clips, e := s.twitch.Clips(ctx, tid, startedAt, endedAt)
	if e != nil {
		return nil, e
	}
	for i := range clips {
		if e = s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM clips WHERE twitch_clip_id=$1)", clips[i].ID).Scan(&clips[i].Saved); e != nil {
			return nil, e
		}
	}
	return clips, nil
}
func (s *Service) Import(ctx context.Context, streamerID string, c twitch.Clip) (string, bool, error) {
	var id string
	e := s.db.QueryRow(ctx, `INSERT INTO clips(streamer_id,twitch_clip_id,title,twitch_url,thumbnail_url,duration,twitch_created_at,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,'saved') ON CONFLICT(twitch_clip_id) DO NOTHING RETURNING id`, streamerID, c.ID, c.Title, c.URL, c.ThumbnailURL, c.Duration, c.CreatedAt).Scan(&id)
	if e == pgx.ErrNoRows {
		e = s.db.QueryRow(ctx, "SELECT id FROM clips WHERE twitch_clip_id=$1", c.ID).Scan(&id)
		return id, false, e
	}
	if e != nil {
		return "", false, e
	}
	return id, true, nil
}
