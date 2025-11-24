-- Evaluation results table (stores LLM-as-a-Judge evaluation outputs)
CREATE TABLE IF NOT EXISTS evaluation_results (
    id UUID,
    job_id Nullable(UUID),
    evaluator_id UUID,
    project_id UUID,
    trace_id UUID,
    span_id Nullable(UUID),

    -- Result data
    score Float64,
    string_value Nullable(String),
    reasoning Nullable(String),
    raw_response String DEFAULT '',

    -- Cost and usage tracking
    prompt_tokens UInt32 DEFAULT 0,
    completion_tokens UInt32 DEFAULT 0,
    cost Decimal64(8) DEFAULT 0,
    latency_ms UInt32 DEFAULT 0,

    -- Status
    status String DEFAULT 'success',
    error_message Nullable(String),

    created_at DateTime64(3) DEFAULT now64(3),

    INDEX idx_job_id job_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_evaluator_id evaluator_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_status status TYPE bloom_filter(0.01) GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (project_id, evaluator_id, created_at, id)
TTL toDateTime(created_at) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

-- Materialized view for evaluation aggregations by evaluator
CREATE MATERIALIZED VIEW IF NOT EXISTS evaluation_daily_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, evaluator_id, date)
AS SELECT
    project_id,
    evaluator_id,
    toDate(created_at) AS date,
    count() AS eval_count,
    sum(cost) AS total_cost,
    sum(prompt_tokens + completion_tokens) AS total_tokens,
    avg(score) AS avg_score,
    avg(latency_ms) AS avg_latency,
    countIf(status = 'success') AS success_count,
    countIf(status = 'error') AS error_count
FROM evaluation_results
GROUP BY project_id, evaluator_id, date;
