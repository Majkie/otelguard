-- OTelGuard ClickHouse Schema
-- Migration 002: Events Table

-- ============================================
-- Events Table
-- Generic events storage for logs, exceptions,
-- custom events, and other observability data
-- ============================================
CREATE TABLE IF NOT EXISTS events (
                                      id UUID,
                                      project_id UUID,
                                      trace_id Nullable(UUID),
    span_id Nullable(UUID),
    session_id Nullable(String),
    user_id Nullable(String),

    -- Event identification
    name String,
    event_type String,  -- renamed from `type` to avoid reserved-word ambiguity: 'log','exception','custom','user_action','system'
    level String DEFAULT 'info',  -- 'debug', 'info', 'warn', 'error', 'fatal'

-- Event content
    message String,
    data String DEFAULT '{}',  -- JSON object for additional structured data

-- Exception-specific fields (nullable for non-exception events)
    exception_type Nullable(String),
    exception_message Nullable(String),
    exception_stacktrace Nullable(String),

    -- Context
    source String DEFAULT '',  -- Component or service that generated the event
    environment String DEFAULT '',
    version String DEFAULT '',

    -- Metadata
    tags Array(String) DEFAULT [],
    attributes Map(String, String) DEFAULT map(),

    -- Timestamps
    timestamp DateTime64(3),
    created_at DateTime64(3) DEFAULT now64(3),

    -- Indexes
    INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_span_id span_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_session_id session_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_user_id user_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_event_type event_type TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_level level TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_name name TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_source source TYPE bloom_filter(0.01) GRANULARITY 4
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMM(timestamp)
    ORDER BY (project_id, timestamp, id)
    TTL toDateTime(timestamp) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- ============================================
-- Events Daily Stats Materialized View
-- ============================================
CREATE MATERIALIZED VIEW IF NOT EXISTS event_daily_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, event_type, level)
AS
SELECT
    project_id,
    toDate(timestamp) AS date,
    event_type,
    level,
    count() AS event_count
FROM events
GROUP BY project_id, date, event_type, level;

-- ============================================
-- Exception Summary Materialized View
-- ============================================
CREATE MATERIALIZED VIEW IF NOT EXISTS exception_summary
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, assumeNotNull(exception_type))
AS
SELECT
    project_id,
    toDate(timestamp) AS date,
    exception_type,
    count() AS exception_count
FROM events
WHERE event_type = 'exception' AND exception_type IS NOT NULL
GROUP BY project_id, date, exception_type;
