// Package videotemplate stores validated, declarative render compositions.
package videotemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Template struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	PreviewAssetID *string            `json:"previewAssetId"`
	PreviewURL     string             `json:"previewUrl,omitempty"`
	IsDefault      bool               `json:"isDefault"`
	ConfigVersion  int                `json:"configVersion"`
	Config         composition.Config `json:"config"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

type Input struct {
	Name           string
	Description    string
	PreviewAssetID *string
	Config         composition.Config
}

type Service struct {
	db      *pgxpool.Pool
	storage media.Storage
}

func New(db *pgxpool.Pool, storage media.Storage) *Service {
	return &Service{db: db, storage: storage}
}

func (s *Service) List(ctx context.Context) ([]Template, error) {
	rows, err := s.db.Query(ctx, `SELECT t.id,t.name,t.description,t.preview_asset_id,t.is_default,t.config_version,t.config,t.created_at,t.updated_at,COALESCE(a.storage_key,'')
		FROM video_templates t LEFT JOIN assets a ON a.id=t.preview_asset_id ORDER BY t.is_default DESC,t.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Template{}
	for rows.Next() {
		item, key, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		if key != "" {
			item.PreviewURL, err = s.storage.PresignGet(ctx, key, 15*time.Minute)
			if err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) Get(ctx context.Context, id string) (Template, error) {
	row := s.db.QueryRow(ctx, `SELECT t.id,t.name,t.description,t.preview_asset_id,t.is_default,t.config_version,t.config,t.created_at,t.updated_at,COALESCE(a.storage_key,'')
		FROM video_templates t LEFT JOIN assets a ON a.id=t.preview_asset_id WHERE t.id=$1`, id)
	item, key, err := scanTemplate(row)
	if err == pgx.ErrNoRows {
		return Template{}, fmt.Errorf("template not found")
	}
	if err != nil {
		return Template{}, err
	}
	if key != "" {
		item.PreviewURL, err = s.storage.PresignGet(ctx, key, 15*time.Minute)
	}
	return item, err
}

func (s *Service) Create(ctx context.Context, in Input) (Template, error) {
	if err := validateInput(in); err != nil {
		return Template{}, err
	}
	configJSON, err := in.Config.JSON()
	if err != nil {
		return Template{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Template{}, err
	}
	defer tx.Rollback(ctx)
	if err = s.ensureAssets(ctx, tx, in.Config); err != nil {
		return Template{}, err
	}
	if err = s.ensurePreview(ctx, tx, in.PreviewAssetID); err != nil {
		return Template{}, err
	}
	var item Template
	err = tx.QueryRow(ctx, `INSERT INTO video_templates(name,description,preview_asset_id,config_version,config) VALUES($1,$2,$3,$4,$5)
		RETURNING id,name,description,preview_asset_id,is_default,config_version,config,created_at,updated_at`, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.PreviewAssetID, in.Config.Version, configJSON).Scan(&item.ID, &item.Name, &item.Description, &item.PreviewAssetID, &item.IsDefault, &item.ConfigVersion, &configJSON, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Template{}, uniqueError(err)
	}
	if item.Config, err = composition.ParseConfig(configJSON); err != nil {
		return Template{}, err
	}
	if cmd, defaultErr := tx.Exec(ctx, `UPDATE video_templates SET is_default=true WHERE id=$1 AND NOT EXISTS (SELECT 1 FROM video_templates WHERE is_default)`, item.ID); defaultErr != nil {
		return Template{}, defaultErr
	} else if cmd.RowsAffected() > 0 {
		item.IsDefault = true
	}
	if err = s.replaceReferences(ctx, tx, item.ID, in.Config.AssetIDs()); err != nil {
		return Template{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Template{}, err
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, id string, in Input) (Template, error) {
	if err := validateInput(in); err != nil {
		return Template{}, err
	}
	configJSON, err := in.Config.JSON()
	if err != nil {
		return Template{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Template{}, err
	}
	defer tx.Rollback(ctx)
	if err = s.ensureAssets(ctx, tx, in.Config); err != nil {
		return Template{}, err
	}
	if err = s.ensurePreview(ctx, tx, in.PreviewAssetID); err != nil {
		return Template{}, err
	}
	var item Template
	err = tx.QueryRow(ctx, `UPDATE video_templates SET name=$2,description=$3,preview_asset_id=$4,config_version=$5,config=$6,updated_at=now() WHERE id=$1
		RETURNING id,name,description,preview_asset_id,is_default,config_version,config,created_at,updated_at`, id, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.PreviewAssetID, in.Config.Version, configJSON).Scan(&item.ID, &item.Name, &item.Description, &item.PreviewAssetID, &item.IsDefault, &item.ConfigVersion, &configJSON, &item.CreatedAt, &item.UpdatedAt)
	if err == pgx.ErrNoRows {
		return Template{}, fmt.Errorf("template not found")
	}
	if err != nil {
		return Template{}, uniqueError(err)
	}
	if item.Config, err = composition.ParseConfig(configJSON); err != nil {
		return Template{}, err
	}
	if err = s.replaceReferences(ctx, tx, item.ID, in.Config.AssetIDs()); err != nil {
		return Template{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Template{}, err
	}
	return item, nil
}

func (s *Service) Duplicate(ctx context.Context, id string) (Template, error) {
	original, err := s.Get(ctx, id)
	if err != nil {
		return Template{}, err
	}
	return s.Create(ctx, Input{Name: original.Name + " — копия", Description: original.Description, PreviewAssetID: original.PreviewAssetID, Config: original.Config})
}

func (s *Service) Delete(ctx context.Context, id string) error {
	var active, isDefault bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM processing_jobs WHERE template_id=$1 AND status IN ('pending','running')),COALESCE((SELECT is_default FROM video_templates WHERE id=$1),false)`, id).Scan(&active, &isDefault); err != nil {
		return err
	}
	if isDefault {
		return fmt.Errorf("default template cannot be deleted; choose another default first")
	}
	if active {
		return fmt.Errorf("template has an active processing job")
	}
	cmd, err := s.db.Exec(ctx, `DELETE FROM video_templates WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}

func (s *Service) SetDefault(ctx context.Context, id string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM video_templates WHERE id=$1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("template not found")
	}
	if _, err = tx.Exec(ctx, `UPDATE video_templates SET is_default=false WHERE is_default`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE video_templates SET is_default=true,updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Snapshot validates and resolves each referenced asset into an immutable S3 key.
// This is the exact point where the template becomes independent from future edits.
func (s *Service) Snapshot(ctx context.Context, id string) ([]byte, error) {
	var templateID, name string
	var raw []byte
	err := s.db.QueryRow(ctx, `SELECT id,name,config FROM video_templates WHERE id=$1`, id).Scan(&templateID, &name, &raw)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, err
	}
	config, err := composition.ParseConfig(raw)
	if err != nil {
		return nil, err
	}
	return s.snapshotConfig(ctx, templateID, name, config)
}

// SnapshotConfig stores an edited render draft without mutating the base template.
func (s *Service) SnapshotConfig(ctx context.Context, id string, config composition.Config) ([]byte, error) {
	var name string
	if err := s.db.QueryRow(ctx, `SELECT name FROM video_templates WHERE id=$1`, id).Scan(&name); err == pgx.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	} else if err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return s.snapshotConfig(ctx, id, name, config)
}

func (s *Service) snapshotConfig(ctx context.Context, id, name string, config composition.Config) ([]byte, error) {
	refs, err := s.assetSnapshots(ctx, config)
	if err != nil {
		return nil, err
	}
	return json.Marshal(composition.Snapshot{Version: composition.CurrentVersion, TemplateID: id, TemplateName: name, Config: config, Assets: refs})
}

// DefaultID is used only to retry a legacy job created before templates existed.
func (s *Service) DefaultID(ctx context.Context) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, `SELECT id FROM video_templates ORDER BY is_default DESC,created_at LIMIT 1`).Scan(&id)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("no video templates exist")
	}
	return id, err
}

func validateInput(in Input) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("template name is required")
	}
	return in.Config.Validate()
}

func (s *Service) ensureAssets(ctx context.Context, tx pgx.Tx, config composition.Config) error {
	for _, id := range config.AssetIDs() {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("asset %s not found", id)
		}
	}
	return nil
}

func (s *Service) ensurePreview(ctx context.Context, tx pgx.Tx, id *string) error {
	if id == nil || *id == "" {
		return nil
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1)`, *id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("preview asset not found")
	}
	return nil
}

func (s *Service) replaceReferences(ctx context.Context, tx pgx.Tx, templateID string, ids []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM template_asset_references WHERE template_id=$1`, templateID); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `INSERT INTO template_asset_references(template_id,asset_id) VALUES($1,$2)`, templateID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) assetSnapshots(ctx context.Context, config composition.Config) ([]composition.AssetSnapshot, error) {
	items := make([]composition.AssetSnapshot, 0)
	for _, id := range config.AssetIDs() {
		var item composition.AssetSnapshot
		err := s.db.QueryRow(ctx, `SELECT id,storage_key,kind::text,mime_type FROM assets WHERE id=$1`, id).Scan(&item.ID, &item.StorageKey, &item.Kind, &item.MIMEType)
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("asset %s not found", id)
		}
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

type templateRow interface{ Scan(...any) error }

func scanTemplate(row templateRow) (Template, string, error) {
	var item Template
	var raw []byte
	var key string
	err := row.Scan(&item.ID, &item.Name, &item.Description, &item.PreviewAssetID, &item.IsDefault, &item.ConfigVersion, &raw, &item.CreatedAt, &item.UpdatedAt, &key)
	if err != nil {
		return Template{}, "", err
	}
	config, err := composition.ParseConfig(raw)
	if err != nil {
		return Template{}, "", fmt.Errorf("invalid saved template %s: %w", item.ID, err)
	}
	item.Config = config
	return item, key, nil
}

func uniqueError(err error) error {
	if strings.Contains(err.Error(), "duplicate key") {
		return fmt.Errorf("a template with this name already exists")
	}
	return err
}
