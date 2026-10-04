package media

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Files struct{ db *pgxpool.Pool }

func NewFiles(db *pgxpool.Pool) *Files { return &Files{db} }
func (f *Files) Upsert(ctx context.Context, clipID, jobID, kind, key, mime string, size int64) error {
	_, e := f.db.Exec(ctx, `INSERT INTO media_files(clip_id,processing_job_id,type,storage_key,mime_type,size,folder_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(storage_key) DO UPDATE SET processing_job_id=EXCLUDED.processing_job_id,mime_type=EXCLUDED.mime_type,size=EXCLUDED.size,folder_id=EXCLUDED.folder_id`, clipID, jobID, kind, key, mime, size, systemFolderID(kind))
	return e
}

func systemFolderID(kind string) string {
	switch kind {
	case "audio":
		return "00000000-0000-0000-0000-000000000101"
	case "subtitle":
		return "00000000-0000-0000-0000-000000000102"
	default:
		return "00000000-0000-0000-0000-000000000103"
	}
}
