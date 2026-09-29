// Package sourcevideo exposes a pinned Assets folder as the long-form video library.
package sourcevideo

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const systemStreamerID = "00000000-0000-0000-0000-000000000001"

type Folder struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parentId"`
	Name     string  `json:"name"`
}
type Video struct {
	ID        string    `json:"id"`
	FolderID  *string   `json:"folderId"`
	Name      string    `json:"name"`
	MIMEType  string    `json:"mimeType"`
	Size      int64     `json:"size"`
	Duration  float64   `json:"duration"`
	URL       string    `json:"url"`
	Cuts      int       `json:"cuts"`
	CreatedAt time.Time `json:"createdAt"`
}
type Library struct {
	FolderID *string  `json:"folderId"`
	Folders  []Folder `json:"folders"`
	Videos   []Video  `json:"videos"`
}
type Service struct {
	db      *pgxpool.Pool
	storage media.Storage
}

func New(db *pgxpool.Pool, storage media.Storage) *Service { return &Service{db: db, storage: storage} }

func (s *Service) List(ctx context.Context) (Library, error) {
	result := Library{Folders: []Folder{}, Videos: []Video{}}
	if err := s.db.QueryRow(ctx, `SELECT folder_id FROM source_video_settings WHERE singleton=true`).Scan(&result.FolderID); err != nil {
		return result, err
	}
	rows, err := s.db.Query(ctx, `SELECT id,parent_id,name FROM asset_folders ORDER BY name`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item Folder
		if err = rows.Scan(&item.ID, &item.ParentID, &item.Name); err != nil {
			rows.Close()
			return result, err
		}
		result.Folders = append(result.Folders, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	if result.FolderID == nil {
		return result, nil
	}
	rows, err = s.db.Query(ctx, `SELECT a.id,a.folder_id,a.name,a.mime_type,a.size,COALESCE((a.metadata->>'duration')::double precision,0),a.storage_key,a.created_at,COUNT(c.id)
		FROM assets a LEFT JOIN clips c ON c.origin_asset_id=a.id WHERE a.folder_id=$1 AND a.kind='video' GROUP BY a.id ORDER BY a.created_at DESC`, *result.FolderID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Video
		var key string
		if err = rows.Scan(&item.ID, &item.FolderID, &item.Name, &item.MIMEType, &item.Size, &item.Duration, &key, &item.CreatedAt, &item.Cuts); err != nil {
			return result, err
		}
		item.URL, err = s.storage.PresignGet(ctx, key, 15*time.Minute)
		if err != nil {
			return result, err
		}
		result.Videos = append(result.Videos, item)
	}
	return result, rows.Err()
}

func (s *Service) PinFolder(ctx context.Context, folderID *string) error {
	if folderID != nil {
		var exists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM asset_folders WHERE id=$1)`, *folderID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("folder not found")
		}
	}
	_, err := s.db.Exec(ctx, `UPDATE source_video_settings SET folder_id=$1,updated_at=now() WHERE singleton=true`, folderID)
	return err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	var key string
	var cuts int
	err := s.db.QueryRow(ctx, `SELECT a.storage_key,COUNT(c.id) FROM assets a LEFT JOIN clips c ON c.source_asset_id=a.id WHERE a.id=$1 AND a.kind='video' GROUP BY a.id`, id).Scan(&key, &cuts)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("video asset not found")
	}
	if err != nil {
		return err
	}
	if cuts > 0 {
		return fmt.Errorf("delete this video's fragments from Pipeline first")
	}
	if err = s.storage.Delete(ctx, key); err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `DELETE FROM assets WHERE id=$1`, id)
	return err
}

func (s *Service) Upload(ctx context.Context, name, contentType string, size int64, body io.Reader) ([]Video, error) {
	var folderID *string
	if err := s.db.QueryRow(ctx, `SELECT folder_id FROM source_video_settings WHERE singleton=true`).Scan(&folderID); err != nil {
		return nil, err
	}
	if folderID == nil {
		return nil, fmt.Errorf("pin an Assets folder first")
	}
	name = cleanName(name)
	if name == "" || size < 1 {
		return nil, fmt.Errorf("video file must not be empty")
	}
	if isZip(name, contentType) {
		return s.uploadZip(ctx, *folderID, body)
	}
	item, err := s.store(ctx, *folderID, name, videoMIME(name, contentType), size, body)
	if err != nil {
		return nil, err
	}
	return []Video{item}, nil
}
func (s *Service) uploadZip(ctx context.Context, folderID string, body io.Reader) ([]Video, error) {
	tmp, err := os.CreateTemp("", "finde-video-*.zip")
	if err != nil {
		return nil, err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err = io.Copy(tmp, body); err != nil {
		tmp.Close()
		return nil, err
	}
	if err = tmp.Close(); err != nil {
		return nil, err
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}
	defer archive.Close()
	result := []Video{}
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() || !isVideoName(entry.Name) {
			continue
		}
		if entry.UncompressedSize64 > 20<<30 {
			return nil, fmt.Errorf("%s is larger than 20 GB", filepath.Base(entry.Name))
		}
		reader, e := entry.Open()
		if e != nil {
			return nil, e
		}
		item, e := s.store(ctx, folderID, cleanName(entry.Name), videoMIME(entry.Name, ""), int64(entry.UncompressedSize64), reader)
		reader.Close()
		if e != nil {
			return nil, e
		}
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("zip archive contains no supported video files")
	}
	return result, nil
}
func (s *Service) store(ctx context.Context, folderID, name, contentType string, size int64, body io.Reader) (Video, error) {
	if !strings.HasPrefix(contentType, "video/") {
		return Video{}, fmt.Errorf("unsupported video type %q", contentType)
	}
	id, err := uuid()
	if err != nil {
		return Video{}, err
	}
	key := "assets/" + id + "/original"
	if err = s.storage.Put(ctx, key, body, contentType); err != nil {
		return Video{}, fmt.Errorf("upload video asset: %w", err)
	}
	var item Video
	err = s.db.QueryRow(ctx, `INSERT INTO assets(id,folder_id,name,kind,mime_type,size,storage_key) VALUES($1,$2,$3,'video',$4,$5,$6) RETURNING id,folder_id,name,mime_type,size,created_at`, id, folderID, name, contentType, size, key).Scan(&item.ID, &item.FolderID, &item.Name, &item.MIMEType, &item.Size, &item.CreatedAt)
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return Video{}, err
	}
	item.URL, err = s.storage.PresignGet(ctx, key, 15*time.Minute)
	return item, err
}
func (s *Service) SetDuration(ctx context.Context, id string, duration float64) error {
	if duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	command, err := s.db.Exec(ctx, `UPDATE assets SET metadata=jsonb_set(metadata,'{duration}',to_jsonb($2::double precision)),updated_at=now() WHERE id=$1 AND kind='video'`, id, duration)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return fmt.Errorf("video asset not found")
	}
	return nil
}
func (s *Service) CreateCut(ctx context.Context, assetID, title string, timeline composition.Timeline) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("cut title is required")
	}
	if len(timeline.Segments) < 1 || len(timeline.Segments) > 100 {
		return "", fmt.Errorf("cut must contain from 1 to 100 segments")
	}
	var duration float64
	err := s.db.QueryRow(ctx, `SELECT COALESCE((metadata->>'duration')::double precision,0) FROM assets WHERE id=$1 AND kind='video'`, assetID).Scan(&duration)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("video asset not found")
	}
	if err != nil {
		return "", err
	}
	for i := range timeline.Segments {
		segment := &timeline.Segments[i]
		segment.Source = "clip"
		segment.AssetID = ""
		segment.ClipID = ""
		segment.SourceDuration = duration
		if segment.Start < 0 || segment.End <= segment.Start || segment.End > duration+0.001 {
			return "", fmt.Errorf("segment %d is invalid", i+1)
		}
	}
	raw, err := json.Marshal(timeline)
	if err != nil {
		return "", err
	}
	id, err := uuid()
	if err != nil {
		return "", err
	}
	var created string
	outputDuration := 0.0
	for _, segment := range timeline.Segments {
		outputDuration += segment.End - segment.Start
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO clips(id,streamer_id,twitch_clip_id,title,twitch_url,duration,status,is_ready_fragment,edit_timeline,source_asset_id,origin_asset_id) VALUES($1,$2,$3,$4,'',$5,'rendering',false,$6,$7,$7) RETURNING id`, id, systemStreamerID, "library:"+id, title, outputDuration, raw, assetID).Scan(&created)
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO processing_jobs(clip_id,type) VALUES($1,'fragment')`, created); err != nil {
		return "", err
	}
	return created, tx.Commit(ctx)
}

func cleanName(v string) string {
	return strings.Trim(strings.TrimSpace(filepath.Base(filepath.ToSlash(v))), ".")
}
func isZip(n, t string) bool {
	return strings.EqualFold(filepath.Ext(n), ".zip") || strings.Contains(strings.ToLower(t), "zip")
}
func isVideoName(n string) bool {
	switch strings.ToLower(filepath.Ext(n)) {
	case ".mp4", ".m4v", ".mov", ".webm", ".mkv", ".avi":
		return true
	}
	return false
}
func videoMIME(n, t string) string {
	t = strings.ToLower(strings.TrimSpace(strings.Split(t, ";")[0]))
	if strings.HasPrefix(t, "video/") {
		return t
	}
	if v := mime.TypeByExtension(strings.ToLower(filepath.Ext(n))); strings.HasPrefix(v, "video/") {
		return v
	}
	switch strings.ToLower(filepath.Ext(n)) {
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	}
	return "video/mp4"
}
func uuid() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32], nil
}
