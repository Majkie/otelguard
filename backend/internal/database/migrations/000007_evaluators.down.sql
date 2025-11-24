-- Drop triggers
DROP TRIGGER IF EXISTS evaluation_jobs_updated_at ON evaluation_jobs;
DROP TRIGGER IF EXISTS evaluators_updated_at ON evaluators;
DROP FUNCTION IF EXISTS update_evaluators_updated_at();

-- Drop tables
DROP TABLE IF EXISTS evaluation_jobs;
DROP TABLE IF EXISTS evaluators;
