-- ============================================
-- User Feedback Tables Migration
-- ============================================

-- Create user_feedback table
CREATE TABLE user_feedback (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(255),
    trace_id VARCHAR(255),
    span_id VARCHAR(255),
    item_type VARCHAR(50) NOT NULL, -- 'trace', 'session', 'span', 'prompt'
    item_id VARCHAR(255) NOT NULL,
    thumbs_up BOOLEAN,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    metadata JSONB DEFAULT '{}',
    user_agent TEXT,
    ip_address TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_user_feedback_project_id ON user_feedback(project_id);
CREATE INDEX idx_user_feedback_user_id ON user_feedback(user_id);
CREATE INDEX idx_user_feedback_session_id ON user_feedback(session_id);
CREATE INDEX idx_user_feedback_trace_id ON user_feedback(trace_id);
CREATE INDEX idx_user_feedback_span_id ON user_feedback(span_id);
CREATE INDEX idx_user_feedback_item_type_item_id ON user_feedback(item_type, item_id);
CREATE INDEX idx_user_feedback_created_at ON user_feedback(created_at);
CREATE INDEX idx_user_feedback_rating ON user_feedback(rating);

-- Unique constraint to prevent duplicate feedback (one feedback per user per item)
-- Allow NULL user_id for anonymous feedback
CREATE UNIQUE INDEX idx_user_feedback_unique_user_item
ON user_feedback(project_id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), item_type, item_id);

-- Create updated_at trigger
CREATE TRIGGER update_user_feedback_updated_at
    BEFORE UPDATE ON user_feedback
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
