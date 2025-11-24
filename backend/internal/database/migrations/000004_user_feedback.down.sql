-- ============================================
-- User Feedback Tables Down Migration
-- ============================================

-- Drop triggers
DROP TRIGGER IF EXISTS update_user_feedback_updated_at ON user_feedback;

-- Drop indexes
DROP INDEX IF EXISTS idx_user_feedback_unique_user_item;
DROP INDEX IF EXISTS idx_user_feedback_rating;
DROP INDEX IF EXISTS idx_user_feedback_created_at;
DROP INDEX IF EXISTS idx_user_feedback_item_type_item_id;
DROP INDEX IF EXISTS idx_user_feedback_span_id;
DROP INDEX IF EXISTS idx_user_feedback_trace_id;
DROP INDEX IF EXISTS idx_user_feedback_session_id;
DROP INDEX IF EXISTS idx_user_feedback_user_id;
DROP INDEX IF EXISTS idx_user_feedback_project_id;

-- Drop table
DROP TABLE IF EXISTS user_feedback;
