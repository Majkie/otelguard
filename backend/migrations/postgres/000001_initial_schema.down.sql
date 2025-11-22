-- OTelGuard PostgreSQL Schema Rollback
-- Rollback initial migration: Drop all core tables

-- Drop triggers first
DROP TRIGGER IF EXISTS update_annotation_queues_updated_at ON annotation_queues;
DROP TRIGGER IF EXISTS update_datasets_updated_at ON datasets;
DROP TRIGGER IF EXISTS update_guardrail_policies_updated_at ON guardrail_policies;
DROP TRIGGER IF EXISTS update_prompts_updated_at ON prompts;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS score_configs;
DROP TABLE IF EXISTS annotation_queues;
DROP TABLE IF EXISTS guardrail_rules;
DROP TABLE IF EXISTS guardrail_policies;
DROP TABLE IF EXISTS dataset_items;
DROP TABLE IF EXISTS datasets;
DROP TABLE IF EXISTS prompt_versions;
DROP TABLE IF EXISTS prompts;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organizations;
