-- Fix agents table schema
-- 1. Correct cost column type if it was wrong (Decimal64(8) -> Decimal(18,8))
ALTER TABLE agents MODIFY COLUMN IF EXISTS cost Decimal(18,8) DEFAULT 0;

-- 2. Add parent_span_id column
ALTER TABLE agents ADD COLUMN IF NOT EXISTS parent_span_id Nullable(UUID);
ALTER TABLE agents ADD INDEX IF NOT EXISTS idx_parent_span_id parent_span_id TYPE bloom_filter GRANULARITY 4;

-- 3. Ensure agent_relationships table exists (if missed previously)
CREATE TABLE IF NOT EXISTS agent_relationships (
    id UUID,
    project_id UUID,
    trace_id UUID,
    source_agent_id UUID,
    target_agent_id UUID,
    relation_type String,  -- delegates_to, calls, responds_to, supervises, collaborates
    timestamp DateTime64(3),
    metadata String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_source_agent_id source_agent_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_target_agent_id target_agent_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_relation_type relation_type TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, trace_id, timestamp, id)
TTL toDate(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
