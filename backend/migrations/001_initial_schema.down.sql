-- 001_initial_schema.down.sql
-- Rollback: drop all tables in reverse dependency order

DROP TABLE IF EXISTS ai_configs CASCADE;
DROP TABLE IF EXISTS live_frame_snapshots CASCADE;
DROP TABLE IF EXISTS live_streams CASCADE;
DROP TABLE IF EXISTS appeals CASCADE;
DROP TABLE IF EXISTS audit_records CASCADE;
DROP TABLE IF EXISTS content_elements CASCADE;
DROP TABLE IF EXISTS contents_live_stream CASCADE;
DROP TABLE IF EXISTS contents_short_video CASCADE;
DROP TABLE IF EXISTS contents_photo CASCADE;
DROP TABLE IF EXISTS tenant_custom_words CASCADE;
DROP TABLE IF EXISTS tenant_audit_levels CASCADE;
DROP TABLE IF EXISTS tenant_audit_rules CASCADE;
DROP TABLE IF EXISTS contents CASCADE;
DROP TABLE IF EXISTS quality_audit_records CASCADE;
DROP TABLE IF EXISTS quality_audit_batches CASCADE;
DROP TABLE IF EXISTS system_configs CASCADE;
DROP TABLE IF EXISTS audit_team_members CASCADE;
DROP TABLE IF EXISTS audit_teams CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
