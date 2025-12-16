-- Add versioning support to guardrail policies

-- Add current_version column to existing guardrail_policies table
ALTER TABLE guardrail_policies
ADD COLUMN current_version INTEGER NOT NULL DEFAULT 1;

-- Create guardrail_policy_versions table
CREATE TABLE guardrail_policy_versions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    policy_id UUID NOT NULL REFERENCES guardrail_policies(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 0,
    triggers JSONB NOT NULL DEFAULT '{}',
    rules JSONB NOT NULL DEFAULT '[]',
    change_notes TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(policy_id, version)
);

CREATE INDEX idx_guardrail_policy_versions_policy_id ON guardrail_policy_versions(policy_id);
CREATE INDEX idx_guardrail_policy_versions_version ON guardrail_policy_versions(policy_id, version);

-- Create initial version snapshots for existing policies
INSERT INTO guardrail_policy_versions (
    id, policy_id, version, name, description, enabled, priority, triggers, rules, created_by, created_at
)
SELECT
    uuidv7(),
    p.id,
    1,
    p.name,
    p.description,
    p.enabled,
    p.priority,
    p.triggers,
    COALESCE(
        (SELECT jsonb_agg(
            jsonb_build_object(
                'id', r.id,
                'type', r.type,
                'config', r.config,
                'action', r.action,
                'actionConfig', r.action_config,
                'orderIndex', r.order_index
            ) ORDER BY r.order_index
        )
        FROM guardrail_rules r
        WHERE r.policy_id = p.id),
        '[]'::jsonb
    ),
    NULL, -- No user tracking for existing policies (created_by is nullable)
    p.created_at
FROM guardrail_policies p;
