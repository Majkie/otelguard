-- Prompts table for versioned prompt storage
CREATE TABLE IF NOT EXISTS prompts (
    id UUID,
    project_id UUID,
    name String,
    description String DEFAULT '',
    template String,
    version Int32,
    variables Array(String) DEFAULT [],
    created_by Nullable(String),
    created_at DateTime64(3),
    is_active UInt8 DEFAULT 1,
    
    INDEX idx_name name TYPE bloom_filter(0.01) GRANULARITY 4
)
ENGINE = ReplacingMergeTree(created_at)
ORDER BY (project_id, id, version)
SETTINGS index_granularity = 8192;
