-- Evaluators table (LLM-as-a-Judge configurations)
CREATE TABLE IF NOT EXISTS evaluators (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    type VARCHAR(50) NOT NULL DEFAULT 'llm_judge',
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(255) NOT NULL,
    template TEXT NOT NULL,
    config JSONB DEFAULT '{}',
    output_type VARCHAR(50) NOT NULL DEFAULT 'numeric',
    min_value DOUBLE PRECISION,
    max_value DOUBLE PRECISION,
    categories TEXT[] DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_evaluators_project_id ON evaluators(project_id);
CREATE INDEX idx_evaluators_type ON evaluators(type);
CREATE INDEX idx_evaluators_enabled ON evaluators(enabled) WHERE deleted_at IS NULL;

-- Evaluation jobs table (async job queue)
CREATE TABLE IF NOT EXISTS evaluation_jobs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    evaluator_id UUID NOT NULL REFERENCES evaluators(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    target_type VARCHAR(50) NOT NULL DEFAULT 'trace',
    target_ids JSONB NOT NULL DEFAULT '[]',
    total_items INTEGER NOT NULL DEFAULT 0,
    completed INTEGER NOT NULL DEFAULT 0,
    failed INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    total_cost DOUBLE PRECISION DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_evaluation_jobs_project_id ON evaluation_jobs(project_id);
CREATE INDEX idx_evaluation_jobs_evaluator_id ON evaluation_jobs(evaluator_id);
CREATE INDEX idx_evaluation_jobs_status ON evaluation_jobs(status);
CREATE INDEX idx_evaluation_jobs_pending ON evaluation_jobs(status, created_at) WHERE status = 'pending';

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_evaluators_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER evaluators_updated_at
    BEFORE UPDATE ON evaluators
    FOR EACH ROW
    EXECUTE FUNCTION update_evaluators_updated_at();

CREATE TRIGGER evaluation_jobs_updated_at
    BEFORE UPDATE ON evaluation_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_evaluators_updated_at();
