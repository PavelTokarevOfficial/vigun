ALTER TABLE video_templates
  ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;

WITH first_template AS (
  SELECT id FROM video_templates ORDER BY created_at LIMIT 1
)
UPDATE video_templates
SET is_default = TRUE
WHERE id = (SELECT id FROM first_template);

CREATE UNIQUE INDEX video_templates_one_default
  ON video_templates (is_default)
  WHERE is_default;

