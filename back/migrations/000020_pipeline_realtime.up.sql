CREATE OR REPLACE FUNCTION notify_pipeline_change() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify(
    'pipeline_events',
    json_build_object(
      'scope', 'pipeline',
      'table', TG_TABLE_NAME,
      'operation', TG_OP
    )::text
  );
  RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER clips_pipeline_notify
AFTER INSERT OR UPDATE OR DELETE ON clips
FOR EACH STATEMENT EXECUTE FUNCTION notify_pipeline_change();

CREATE TRIGGER processing_jobs_pipeline_notify
AFTER INSERT OR UPDATE OR DELETE ON processing_jobs
FOR EACH STATEMENT EXECUTE FUNCTION notify_pipeline_change();

CREATE TRIGGER media_files_pipeline_notify
AFTER INSERT OR UPDATE OR DELETE ON media_files
FOR EACH STATEMENT EXECUTE FUNCTION notify_pipeline_change();
