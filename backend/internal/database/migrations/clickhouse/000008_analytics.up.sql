-- Minute-level trace statistics for real-time dashboards
CREATE MATERIALIZED VIEW IF NOT EXISTS trace_minute_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(minute)
ORDER BY (project_id, minute, model)
AS SELECT
    project_id,
    toStartOfMinute(start_time) AS minute,
    model,
    count() AS trace_count,
    sum(latency_ms) AS total_latency_ms,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost
FROM traces
GROUP BY project_id, minute, model;
