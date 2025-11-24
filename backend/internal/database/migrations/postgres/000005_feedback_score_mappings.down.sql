-- ============================================
-- Feedback Score Mappings Down Migration
-- ============================================

-- Drop triggers
DROP TRIGGER IF EXISTS update_feedback_score_mappings_updated_at ON feedback_score_mappings;

-- Drop indexes
DROP INDEX IF EXISTS idx_feedback_score_mappings_enabled;
DROP INDEX IF EXISTS idx_feedback_score_mappings_item_type;
DROP INDEX IF EXISTS idx_feedback_score_mappings_project_id;

-- Drop table
DROP TABLE IF EXISTS feedback_score_mappings;
