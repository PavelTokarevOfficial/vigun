DROP TRIGGER IF EXISTS media_files_pipeline_notify ON media_files;
DROP TRIGGER IF EXISTS processing_jobs_pipeline_notify ON processing_jobs;
DROP TRIGGER IF EXISTS clips_pipeline_notify ON clips;
DROP FUNCTION IF EXISTS notify_pipeline_change();
