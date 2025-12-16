-- ============================================
-- Annotation Tables Migration
-- ============================================

-- Extend annotation_queues table with additional configuration fields
ALTER TABLE annotation_queues ADD COLUMN IF NOT EXISTS config JSONB DEFAULT '{}';
ALTER TABLE annotation_queues ADD COLUMN IF NOT EXISTS item_source VARCHAR(100) DEFAULT 'manual';
ALTER TABLE annotation_queues ADD COLUMN IF NOT EXISTS item_source_config JSONB DEFAULT '{}';
ALTER TABLE annotation_queues ADD COLUMN IF NOT EXISTS assignment_strategy VARCHAR(50) DEFAULT 'round_robin';
ALTER TABLE annotation_queues ADD COLUMN IF NOT EXISTS max_annotations_per_item INTEGER DEFAULT 1;
ALTER TABLE annotation_queues ADD COLUMN IF NOT EXISTS instructions TEXT;
ALTER TABLE annotation_queues ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Create annotation_queue_items table
CREATE TABLE annotation_queue_items (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    queue_id UUID NOT NULL REFERENCES annotation_queues(id) ON DELETE CASCADE,
    item_type VARCHAR(50) NOT NULL, -- 'trace', 'span', 'prompt', 'custom'
    item_id VARCHAR(255) NOT NULL, -- ID of the item being annotated (trace_id, span_id, etc.)
    item_data JSONB, -- Optional: store item data for quick access
    metadata JSONB DEFAULT '{}', -- Additional metadata
    priority INTEGER DEFAULT 0, -- Higher priority items get assigned first
    max_annotations INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_annotation_queue_items_queue_id ON annotation_queue_items(queue_id);
CREATE INDEX idx_annotation_queue_items_item_type_item_id ON annotation_queue_items(item_type, item_id);
CREATE INDEX idx_annotation_queue_items_priority ON annotation_queue_items(priority DESC);

-- Create annotation_assignments table
CREATE TABLE annotation_assignments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    queue_item_id UUID NOT NULL REFERENCES annotation_queue_items(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'assigned', -- 'assigned', 'in_progress', 'completed', 'skipped'
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    skipped_at TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(queue_item_id, user_id) -- One assignment per user per item
);

CREATE INDEX idx_annotation_assignments_queue_item_id ON annotation_assignments(queue_item_id);
CREATE INDEX idx_annotation_assignments_user_id ON annotation_assignments(user_id);
CREATE INDEX idx_annotation_assignments_status ON annotation_assignments(status);

-- Create annotations table
CREATE TABLE annotations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    assignment_id UUID NOT NULL REFERENCES annotation_assignments(id) ON DELETE CASCADE,
    queue_id UUID NOT NULL REFERENCES annotation_queues(id) ON DELETE CASCADE,
    queue_item_id UUID NOT NULL REFERENCES annotation_queue_items(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scores JSONB DEFAULT '{}', -- Key-value pairs of score names to values
    labels TEXT[] DEFAULT '{}', -- Categorical labels
    notes TEXT,
    confidence_score DECIMAL(3,2), -- 0.00 to 1.00
    annotation_time INTERVAL, -- How long it took to annotate
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_annotations_assignment_id ON annotations(assignment_id);
CREATE INDEX idx_annotations_queue_id ON annotations(queue_id);
CREATE INDEX idx_annotations_queue_item_id ON annotations(queue_item_id);
CREATE INDEX idx_annotations_user_id ON annotations(user_id);
CREATE INDEX idx_annotations_created_at ON annotations(created_at);

-- Create inter_annotator_agreements table for tracking agreement metrics
CREATE TABLE inter_annotator_agreements (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    queue_id UUID NOT NULL REFERENCES annotation_queues(id) ON DELETE CASCADE,
    queue_item_id UUID NOT NULL REFERENCES annotation_queue_items(id) ON DELETE CASCADE,
    score_config_name VARCHAR(255) NOT NULL,
    agreement_type VARCHAR(50) NOT NULL, -- 'kappa', 'percentage', 'correlation'
    agreement_value DECIMAL(5,4), -- Agreement value (e.g., 0.85 for 85%)
    annotator_count INTEGER NOT NULL,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(queue_id, queue_item_id, score_config_name, agreement_type)
);

CREATE INDEX idx_inter_annotator_agreements_queue_id ON inter_annotator_agreements(queue_id);
CREATE INDEX idx_inter_annotator_agreements_queue_item_id ON inter_annotator_agreements(queue_item_id);

-- Create updated_at triggers
CREATE TRIGGER update_annotation_queue_items_updated_at
    BEFORE UPDATE ON annotation_queue_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_annotation_assignments_updated_at
    BEFORE UPDATE ON annotation_assignments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_annotations_updated_at
    BEFORE UPDATE ON annotations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
