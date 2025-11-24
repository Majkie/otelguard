-- ============================================
-- Feedback Score Mappings Migration
-- ============================================

-- Create feedback_score_mappings table
CREATE TABLE feedback_score_mappings (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    item_type VARCHAR(50) NOT NULL, -- 'trace', 'session', 'span', 'prompt'
    enabled BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}', -- Configuration for mapping rules
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_feedback_score_mappings_project_id ON feedback_score_mappings(project_id);
CREATE INDEX idx_feedback_score_mappings_item_type ON feedback_score_mappings(item_type);
CREATE INDEX idx_feedback_score_mappings_enabled ON feedback_score_mappings(enabled);

-- Create updated_at trigger
CREATE TRIGGER update_feedback_score_mappings_updated_at
    BEFORE UPDATE ON feedback_score_mappings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
