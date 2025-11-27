-- Alert rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT true,

    -- Metric to monitor
    metric_type VARCHAR(100) NOT NULL, -- 'latency', 'cost', 'error_rate', 'token_count', 'custom'
    metric_field VARCHAR(100), -- For custom metrics

    -- Condition
    condition_type VARCHAR(50) NOT NULL, -- 'threshold', 'anomaly', 'percentage_change'
    operator VARCHAR(20) NOT NULL, -- 'gt', 'lt', 'gte', 'lte', 'eq', 'ne'
    threshold_value DECIMAL(20, 4),

    -- Time window
    window_duration INTEGER NOT NULL DEFAULT 300, -- seconds
    evaluation_frequency INTEGER NOT NULL DEFAULT 60, -- seconds

    -- Filters
    filters JSONB DEFAULT '{}', -- { "model": "gpt-4", "user_id": "abc", etc }

    -- Notification settings
    notification_channels TEXT[] DEFAULT '{}', -- ['email:user@example.com', 'slack:#alerts', 'webhook:http://...']
    notification_message TEXT,

    -- Escalation
    escalation_policy_id UUID,

    -- Grouping and deduplication
    group_by TEXT[] DEFAULT '{}', -- ['model', 'user_id', etc]
    group_wait INTEGER DEFAULT 30, -- seconds before sending grouped alert
    repeat_interval INTEGER DEFAULT 3600, -- seconds before re-sending same alert

    -- Metadata
    severity VARCHAR(20) DEFAULT 'warning', -- 'info', 'warning', 'error', 'critical'
    tags TEXT[] DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID REFERENCES users(id)
);

CREATE INDEX idx_alert_rules_project_id ON alert_rules(project_id);
CREATE INDEX idx_alert_rules_enabled ON alert_rules(enabled) WHERE enabled = true;

-- Alert history table
CREATE TABLE IF NOT EXISTS alert_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    -- Alert details
    status VARCHAR(50) NOT NULL, -- 'firing', 'resolved', 'acknowledged'
    severity VARCHAR(20) NOT NULL,

    -- Values
    metric_value DECIMAL(20, 4),
    threshold_value DECIMAL(20, 4),

    -- Time
    fired_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by UUID REFERENCES users(id),

    -- Grouping
    fingerprint VARCHAR(255) NOT NULL, -- Hash of alert_rule_id + group_by values
    group_labels JSONB DEFAULT '{}',

    -- Notification
    notification_sent BOOLEAN DEFAULT false,
    notification_channels TEXT[],
    notification_error TEXT,

    -- Additional context
    message TEXT,
    annotations JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alert_history_alert_rule_id ON alert_history(alert_rule_id);
CREATE INDEX idx_alert_history_project_id ON alert_history(project_id);
CREATE INDEX idx_alert_history_status ON alert_history(status);
CREATE INDEX idx_alert_history_fingerprint ON alert_history(fingerprint);
CREATE INDEX idx_alert_history_fired_at ON alert_history(fired_at DESC);

-- Alert escalation policies table
CREATE TABLE IF NOT EXISTS alert_escalation_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Escalation steps (array of configs)
    steps JSONB NOT NULL DEFAULT '[]', -- [{"delay": 300, "channels": ["email:..."]}, ...]

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alert_escalation_policies_project_id ON alert_escalation_policies(project_id);

-- Alert notification log table
CREATE TABLE IF NOT EXISTS alert_notification_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_history_id UUID NOT NULL REFERENCES alert_history(id) ON DELETE CASCADE,

    channel_type VARCHAR(50) NOT NULL, -- 'email', 'slack', 'webhook'
    channel_target TEXT NOT NULL,

    status VARCHAR(50) NOT NULL, -- 'sent', 'failed', 'pending'
    error_message TEXT,

    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alert_notification_log_alert_history_id ON alert_notification_log(alert_history_id);
CREATE INDEX idx_alert_notification_log_sent_at ON alert_notification_log(sent_at DESC);

-- Updated_at triggers
CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alert_history_updated_at BEFORE UPDATE ON alert_history
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alert_escalation_policies_updated_at BEFORE UPDATE ON alert_escalation_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
