INSERT INTO processing_jobs(clip_id, type)
SELECT c.id, 'download'
FROM clips c
WHERE c.status = 'saved'
  AND NOT EXISTS (
    SELECT 1
    FROM processing_jobs j
    WHERE j.clip_id = c.id
      AND j.type = 'download'
      AND j.status IN ('pending', 'running')
  )
ON CONFLICT DO NOTHING;

UPDATE clips c
SET status = 'downloading', error = NULL, updated_at = now()
WHERE c.status = 'saved'
  AND EXISTS (
    SELECT 1
    FROM processing_jobs j
    WHERE j.clip_id = c.id
      AND j.type = 'download'
      AND j.status IN ('pending', 'running')
  );
