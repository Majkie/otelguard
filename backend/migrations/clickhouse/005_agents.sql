-- Agents table - stores identified agents within multi-agent systems
CREATE TABLE IF NOT EXISTS agents (
    id UUID,
    project_id UUID,
    trace_id UUID,
    span_id UUID,
    parent_agent_id Nullable(UUID),
    name String,
    agent_type String DEFAULT 'custom',  -- orchestrator, worker, tool_caller, planner, executor, reviewer, custom
    role String DEFAULT '',
    model Nullable(String),
    system_prompt Nullable(String),
    start_time DateTime64(3),
    end_time DateTime64(3),
    latency_ms UInt32,
    total_tokens UInt32 DEFAULT 0,
    cost Decimal64(8) DEFAULT 0,
    status String DEFAULT 'success',  -- running, success, error, timeout
    error_message Nullable(String),
    metadata String DEFAULT '{}',
    tags Array(String) DEFAULT [],
    created_at DateTime64(3) DEFAULT now64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_span_id span_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_parent_agent_id parent_agent_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_agent_type agent_type TYPE bloom_filter GRANULARITY 4,
    INDEX idx_status status TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (project_id, trace_id, start_time, id)
TTL toDate(start_time) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Agent relationships table - tracks relationships between agents
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

-- Tool calls table - tracks tool invocations by agents
CREATE TABLE IF NOT EXISTS tool_calls (
    id UUID,
    project_id UUID,
    trace_id UUID,
    span_id UUID,
    agent_id Nullable(UUID),
    name String,
    description String DEFAULT '',
    input String,
    output String,
    start_time DateTime64(3),
    end_time DateTime64(3),
    latency_ms UInt32,
    status String DEFAULT 'success',  -- success, error, timeout, pending
    error_message Nullable(String),
    retry_count Int32 DEFAULT 0,
    metadata String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_span_id span_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_agent_id agent_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_name name TYPE bloom_filter GRANULARITY 4,
    INDEX idx_status status TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (project_id, trace_id, start_time, id)
TTL toDate(start_time) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Agent messages table - stores messages between agents
CREATE TABLE IF NOT EXISTS agent_messages (
    id UUID,
    project_id UUID,
    trace_id UUID,
    span_id Nullable(UUID),
    from_agent_id UUID,
    to_agent_id UUID,
    message_type String,    -- request, response, notification, broadcast
    role String,            -- user, assistant, system, function, tool
    content String,
    content_type String DEFAULT 'text',  -- text, json, tool_call, tool_result
    sequence_num Int32,
    parent_msg_id Nullable(UUID),
    token_count UInt32 DEFAULT 0,
    timestamp DateTime64(3),
    metadata String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_from_agent_id from_agent_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_to_agent_id to_agent_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_message_type message_type TYPE bloom_filter GRANULARITY 4,
    INDEX idx_role role TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, trace_id, sequence_num, id)
TTL toDate(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Agent states table - stores snapshots of agent state
CREATE TABLE IF NOT EXISTS agent_states (
    id UUID,
    project_id UUID,
    trace_id UUID,
    agent_id UUID,
    span_id Nullable(UUID),
    sequence_num Int32,
    state String,          -- initializing, planning, executing, waiting, thinking, completed, failed
    variables String DEFAULT '{}',
    memory String DEFAULT '{}',
    plan String DEFAULT '',
    reasoning String DEFAULT '',
    timestamp DateTime64(3),
    metadata String DEFAULT '{}',
    created_at DateTime64(3) DEFAULT now64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_agent_id agent_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_state state TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (project_id, trace_id, agent_id, sequence_num, id)
TTL toDate(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Materialized view for agent statistics per trace
CREATE MATERIALIZED VIEW IF NOT EXISTS agent_trace_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, trace_id, date)
AS SELECT
    project_id,
    trace_id,
    toDate(start_time) AS date,
    count() AS agent_count,
    countIf(agent_type = 'orchestrator') AS orchestrator_count,
    countIf(agent_type = 'worker') AS worker_count,
    countIf(agent_type = 'tool_caller') AS tool_caller_count,
    sum(latency_ms) AS total_latency_ms,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost,
    countIf(status = 'error') AS error_count,
    max(latency_ms) AS max_latency_ms
FROM agents
GROUP BY project_id, trace_id, date;

-- Materialized view for tool call statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS tool_call_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, name)
AS SELECT
    project_id,
    toDate(start_time) AS date,
    name,
    count() AS call_count,
    sum(latency_ms) AS total_latency_ms,
    avg(latency_ms) AS avg_latency_ms,
    countIf(status = 'success') AS success_count,
    countIf(status = 'error') AS error_count,
    sum(retry_count) AS total_retries
FROM tool_calls
GROUP BY project_id, date, name;

-- Materialized view for agent message flow
CREATE MATERIALIZED VIEW IF NOT EXISTS agent_message_flow
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, trace_id, from_agent_id, to_agent_id, date)
AS SELECT
    project_id,
    trace_id,
    from_agent_id,
    to_agent_id,
    toDate(timestamp) AS date,
    count() AS message_count,
    sum(token_count) AS total_tokens,
    countIf(message_type = 'request') AS request_count,
    countIf(message_type = 'response') AS response_count
FROM agent_messages
GROUP BY project_id, trace_id, from_agent_id, to_agent_id, date;
