-- Remove guardrail policy versioning

DROP TABLE IF EXISTS guardrail_policy_versions;

ALTER TABLE guardrail_policies
DROP COLUMN IF EXISTS current_version;
