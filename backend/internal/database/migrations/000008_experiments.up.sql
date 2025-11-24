-- Experiments tables for dataset evaluation
-- Migration: 000008

-- ============================================
-- Experiments
-- ============================================
CREATE TABLE experiments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    dataset_id UUID NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_experiments_project_id ON experiments(project_id);
CREATE INDEX idx_experiments_dataset_id ON experiments(dataset_id);
CREATE INDEX idx_experiments_status ON experiments(status);

-- ============================================
-- Experiment Runs
-- ============================================
CREATE TABLE experiment_runs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    run_number INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    total_items INTEGER DEFAULT 0,
    completed_items INTEGER DEFAULT 0,
    failed_items INTEGER DEFAULT 0,
    total_cost DECIMAL(10, 6) DEFAULT 0,
    total_latency_ms BIGINT DEFAULT 0,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(experiment_id, run_number)
);

CREATE INDEX idx_experiment_runs_experiment_id ON experiment_runs(experiment_id);
CREATE INDEX idx_experiment_runs_status ON experiment_runs(status);

-- ============================================
-- Experiment Results
-- ============================================
CREATE TABLE experiment_results (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    run_id UUID NOT NULL REFERENCES experiment_runs(id) ON DELETE CASCADE,
    dataset_item_id UUID NOT NULL REFERENCES dataset_items(id) ON DELETE CASCADE,
    trace_id VARCHAR(255),
    output JSONB,
    scores JSONB DEFAULT '{}',
    latency_ms BIGINT DEFAULT 0,
    tokens_used INTEGER DEFAULT 0,
    cost DECIMAL(10, 6) DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'success',
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_experiment_results_run_id ON experiment_results(run_id);
CREATE INDEX idx_experiment_results_dataset_item_id ON experiment_results(dataset_item_id);
CREATE INDEX idx_experiment_results_trace_id ON experiment_results(trace_id);

-- ============================================
-- Updated At Triggers
-- ============================================
CREATE TRIGGER update_experiments_updated_at
    BEFORE UPDATE ON experiments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
