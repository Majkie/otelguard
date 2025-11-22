-- High-cardinality attribute storage for traces and spans
CREATE TABLE IF NOT EXISTS trace_attributes (
    trace_id UUID,
    span_id String DEFAULT '',
    project_id UUID,
    key String,
    value_type String, -- string, int, float, bool, json
    string_value String DEFAULT '',
    int_value Int64 DEFAULT 0,
    float_value Float64 DEFAULT 0,
    bool_value Bool DEFAULT false,
    timestamp DateTime64(3),

    INDEX idx_key key TYPE bloom_filter GRANULARITY 4,
    INDEX idx_value_type value_type TYPE set(5) GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, trace_id, key, timestamp)
TTL timestamp + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Materialized view for attribute key statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS attribute_key_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, key, date)
AS SELECT
    project_id,
    key,
    toDate(timestamp) AS date,
    count() AS usage_count,
    uniqExact(trace_id) AS unique_traces
FROM trace_attributes
GROUP BY project_id, key, date;
