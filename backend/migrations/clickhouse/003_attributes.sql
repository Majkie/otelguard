-- High-cardinality attribute storage for traces and spans
CREATE TABLE IF NOT EXISTS trace_attributes (
                                                trace_id UUID,
                                                span_id Nullable(UUID),
    project_id UUID,
    key String,
    value_type String, -- string, int, float, bool, json
    string_value String DEFAULT '',
    int_value Int64 DEFAULT 0,
    float_value Float64 DEFAULT 0,
    bool_value UInt8 DEFAULT 0,
    timestamp DateTime64(3),

    INDEX idx_key key TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_value_type value_type TYPE bloom_filter(0.01) GRANULARITY 4
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMM(timestamp)
    ORDER BY (project_id, trace_id, key, timestamp)
    TTL toDateTime(timestamp) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- Materialized view for attribute key statistics
-- Use AggregatingMergeTree + stateful aggregates for correctness (uniqExactState)
CREATE MATERIALIZED VIEW IF NOT EXISTS attribute_key_stats
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, key, date)
AS
SELECT
    project_id,
    key,
    toDate(timestamp) AS date,
    countState() AS usage_count_state,
    uniqExactState(trace_id) AS unique_traces_state
FROM trace_attributes
GROUP BY project_id, key, date;
