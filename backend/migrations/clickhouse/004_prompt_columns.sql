-- Add prompt tracking columns to traces table
-- Migration: Add prompt_id and prompt_version for linking traces to prompt versions

-- Add columns to existing traces table
ALTER TABLE traces ADD COLUMN IF NOT EXISTS prompt_id Nullable(UUID);
ALTER TABLE traces ADD COLUMN IF NOT EXISTS prompt_version Nullable(Int32);

-- Add bloom filter indexes for efficient filtering
ALTER TABLE traces ADD INDEX IF NOT EXISTS idx_prompt_id prompt_id TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE traces ADD INDEX IF NOT EXISTS idx_prompt_version prompt_version TYPE bloom_filter(0.01) GRANULARITY 4;

-- Update materialized views to include prompt aggregations
CREATE MATERIALIZED VIEW IF NOT EXISTS prompt_daily_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, prompt_id, prompt_version, model)
AS
SELECT
    project_id,
    toDate(start_time) AS date,
    prompt_id,
    prompt_version,
    model,
    count() AS trace_count,
    sum(latency_ms) AS total_latency_ms,
    avg(latency_ms) AS avg_latency_ms,
    sum(total_tokens) AS total_tokens,
    avg(total_tokens) AS avg_tokens,
    sum(cost) AS total_cost,
    avg(cost) AS avg_cost,
    countIf(status <> 'success') AS error_count
FROM traces
WHERE prompt_id IS NOT NULL
GROUP BY project_id, date, prompt_id, prompt_version, model;

-- Update session stats to include prompt information
CREATE MATERIALIZED VIEW IF NOT EXISTS prompt_session_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(first_seen)
ORDER BY (project_id, assumeNotNull(session_id), prompt_id, prompt_version)
AS
SELECT
    project_id,
    session_id,
    prompt_id,
    prompt_version,
    min(start_time) AS first_seen,
    max(end_time) AS last_seen,
    count() AS trace_count,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost,
    avg(latency_ms) AS avg_latency_ms
FROM traces
WHERE session_id IS NOT NULL AND prompt_id IS NOT NULL
GROUP BY project_id, session_id, prompt_id, prompt_version;
