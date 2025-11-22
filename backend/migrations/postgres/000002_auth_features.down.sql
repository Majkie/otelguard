-- OTelGuard PostgreSQL Schema Rollback
-- Rollback migration 2: Authentication and authorization features

DROP TRIGGER IF EXISTS update_project_members_updated_at ON project_members;

DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS password_reset_tokens;
