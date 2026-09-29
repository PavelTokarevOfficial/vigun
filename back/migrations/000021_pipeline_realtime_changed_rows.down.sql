DROP TRIGGER IF EXISTS media_files_pipeline_notify ON media_files;
DROP TRIGGER IF EXISTS processing_jobs_pipeline_notify ON processing_jobs;
DROP TRIGGER IF EXISTS clips_pipeline_notify ON clips;

CREATE TRIGGER clips_pipeline_notify
AFTER INSERT OR UPDATE OR DELETE ON clips
FOR EACH STATEMENT EXECUTE FUNCTION notify_pipeline_change();

CREATE TRIGGER processing_jobs_pipeline_notify
AFTER INSERT OR UPDATE OR DELETE ON processing_jobs
FOR EACH STATEMENT EXECUTE FUNCTION notify_pipeline_change();

CREATE TRIGGER media_files_pipeline_notify
AFTER INSERT OR UPDATE OR DELETE ON media_files
FOR EACH STATEMENT EXECUTE FUNCTION notify_pipeline_change();
