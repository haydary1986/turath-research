-- Reverse of 000001_init_schema.up.sql

DROP TABLE IF EXISTS ai_analyses CASCADE;
DROP TABLE IF EXISTS sync_jobs CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS department_metrics_monthly CASCADE;
DROP TABLE IF EXISTS researcher_metrics CASCADE;
DROP TABLE IF EXISTS publication_metrics CASCADE;
DROP TABLE IF EXISTS publication_fields CASCADE;
DROP TABLE IF EXISTS publication_authors CASCADE;
DROP TABLE IF EXISTS publications CASCADE;
DROP TABLE IF EXISTS research_fields CASCADE;
DROP TABLE IF EXISTS journals CASCADE;
DROP TABLE IF EXISTS researchers CASCADE;
DROP TABLE IF EXISTS departments CASCADE;
DROP TABLE IF EXISTS colleges CASCADE;
DROP TABLE IF EXISTS institutions CASCADE;

DROP TYPE IF EXISTS analysis_status;
DROP TYPE IF EXISTS analysis_target;
DROP TYPE IF EXISTS analysis_type;
DROP TYPE IF EXISTS sync_status;
DROP TYPE IF EXISTS sync_target_type;
DROP TYPE IF EXISTS sync_job_type;
DROP TYPE IF EXISTS metric_source;
DROP TYPE IF EXISTS preferred_language;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS journal_ranking;
DROP TYPE IF EXISTS publication_status;
DROP TYPE IF EXISTS publication_type;
DROP TYPE IF EXISTS researcher_status;

DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS "uuid-ossp";
