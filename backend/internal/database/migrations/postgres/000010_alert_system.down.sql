-- Drop triggers
DROP TRIGGER IF EXISTS update_alert_escalation_policies_updated_at ON alert_escalation_policies;
DROP TRIGGER IF EXISTS update_alert_history_updated_at ON alert_history;
DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;

-- Drop tables in reverse order
DROP TABLE IF EXISTS alert_notification_log;
DROP TABLE IF EXISTS alert_escalation_policies;
DROP TABLE IF EXISTS alert_history;
DROP TABLE IF EXISTS alert_rules;
