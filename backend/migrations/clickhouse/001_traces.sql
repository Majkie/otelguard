-- OTelGuard ClickHouse Schema
-- Initial migration: Traces, Spans, Scores, and Guardrail Events

-- ============================================
-- Traces Table
-- ============================================
CREATE TABLE IF NOT EXISTS traces (
                                      id UUID,
                                      project_id UUID,
                                      session_id Nullable(String),
    user_id Nullable(String),
    name String,
    input String,
    output String,
    metadata String DEFAULT '{}',
    start_time DateTime64(3),
    end_time DateTime64(3),
    latency_ms UInt32,
    total_tokens UInt32 DEFAULT 0,
    prompt_tokens UInt32 DEFAULT 0,
    completion_tokens UInt32 DEFAULT 0,
    cost Decimal(18,8) DEFAULT 0,
    model String DEFAULT '',
    tags Array(String) DEFAULT [],
    status String DEFAULT 'success',
    error_message Nullable(String),

    INDEX idx_session_id session_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_user_id user_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_model model TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_status status TYPE bloom_filter(0.01) GRANULARITY 4
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMM(start_time)
    ORDER BY (project_id, start_time, id)
    TTL toDateTime(start_time) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- ============================================
-- Spans Table
-- ============================================
CREATE TABLE IF NOT EXISTS spans (
                                     id UUID,
                                     trace_id UUID,
                                     parent_span_id Nullable(UUID),
    project_id UUID,
    name String,
    span_type String,  -- renamed from `type` to avoid any ambiguity; values: 'llm','retrieval','tool','agent','embedding','custom'
    input String,
    output String,
    metadata String DEFAULT '{}',
    start_time DateTime64(3),
    end_time DateTime64(3),
    latency_ms UInt32,
    tokens UInt32 DEFAULT 0,
    cost Decimal(18,8) DEFAULT 0,
    model Nullable(String),
    status String DEFAULT 'success',
    error_message Nullable(String),

    INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_span_type span_type TYPE bloom_filter(0.01) GRANULARITY 4
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMM(start_time)
    ORDER BY (project_id, trace_id, start_time, id)
    TTL toDateTime(start_time) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- ============================================
-- Scores Table
-- ============================================
CREATE TABLE IF NOT EXISTS scores (
                                      id UUID,
                                      project_id UUID,
                                      trace_id UUID,
                                      span_id Nullable(UUID),
    name String,
    value Float64,
    string_value Nullable(String),
    data_type String,  -- 'numeric', 'boolean', 'categorical'
    source String,     -- 'api', 'llm_judge', 'human', 'user_feedback'
    config_id Nullable(UUID),
    comment Nullable(String),
    created_at DateTime64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_name name TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_source source TYPE bloom_filter(0.01) GRANULARITY 4
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMM(created_at)
    ORDER BY (project_id, trace_id, created_at, id)
    TTL toDateTime(created_at) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- ============================================
-- Guardrail Events Table
-- ============================================
CREATE TABLE IF NOT EXISTS guardrail_events (
                                                id UUID,
                                                project_id UUID,
                                                trace_id Nullable(UUID),
    span_id Nullable(UUID),
    policy_id UUID,
    rule_id UUID,
    rule_type String,
    triggered UInt8 DEFAULT 0, -- use UInt8 instead of Bool for better index behavior; 0/1 values
    action String,
    action_taken UInt8 DEFAULT 0,
    input_text String,
    output_text Nullable(String),
    detection_result String,
    latency_ms UInt32,
    created_at DateTime64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_policy_id policy_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_triggered triggered TYPE minmax() GRANULARITY 4
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMM(created_at)
    ORDER BY (project_id, created_at, id)
    TTL toDateTime(created_at) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- ============================================
-- Metrics Table (for custom metrics)
-- ============================================
CREATE TABLE IF NOT EXISTS metrics (
                                       id UUID,
                                       project_id UUID,
                                       trace_id Nullable(UUID),
    span_id Nullable(UUID),
    name String,
    value Float64,
    unit String DEFAULT '',
    tags Map(String, String) DEFAULT map(),
    created_at DateTime64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_name name TYPE bloom_filter(0.01) GRANULARITY 4
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMM(created_at)
    ORDER BY (project_id, name, created_at, id)
    TTL toDateTime(created_at) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- ============================================
-- Materialized Views for Aggregations
-- Note: SummingMergeTree is fine for simple sums. For more advanced unique counts consider AggregatingMergeTree + state functions.
-- ============================================

-- Daily trace statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS trace_daily_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, model)
AS
SELECT
    project_id,
    toDate(start_time) AS date,
    model,
    count() AS trace_count,
    sum(latency_ms) AS total_latency_ms,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost,
    countIf(status <> 'success') AS error_count
FROM traces
GROUP BY project_id, date, model;

-- Hourly trace statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS trace_hourly_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (project_id, hour, model)
AS
SELECT
    project_id,
    toStartOfHour(start_time) AS hour,
    model,
    count() AS trace_count,
    sum(latency_ms) AS total_latency_ms,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost,
    countIf(status <> 'success') AS error_count
FROM traces
GROUP BY project_id, hour, model;

-- Session aggregations
CREATE MATERIALIZED VIEW IF NOT EXISTS session_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(first_seen)
ORDER BY (project_id, assumeNotNull(session_id))
AS
SELECT
    project_id,
    session_id,
    min(start_time) AS first_seen,
    max(end_time) AS last_seen,
    count() AS trace_count,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost
FROM traces
WHERE session_id IS NOT NULL
GROUP BY project_id, session_id;

-- User aggregations
CREATE MATERIALIZED VIEW IF NOT EXISTS user_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(first_seen)
ORDER BY (project_id, assumeNotNull(user_id))
AS
SELECT
    project_id,
    user_id,
    min(start_time) AS first_seen,
    max(end_time) AS last_seen,
    count() AS trace_count,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost
FROM traces
WHERE user_id IS NOT NULL
GROUP BY project_id, user_id;

-- Guardrail daily stats
CREATE MATERIALIZED VIEW IF NOT EXISTS guardrail_daily_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, policy_id, rule_type)
AS
SELECT
    project_id,
    toDate(created_at) AS date,
    policy_id,
    rule_type,
    count() AS evaluation_count,
    countIf(triggered = 1) AS trigger_count,
    countIf(action_taken = 1) AS action_count,
    sum(latency_ms) AS total_latency_ms
FROM guardrail_events
GROUP BY project_id, date, policy_id, rule_type;
