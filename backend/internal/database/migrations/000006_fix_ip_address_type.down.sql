-- Revert ip_address column type back to INET

-- Revert user_feedback table
ALTER TABLE user_feedback ALTER COLUMN ip_address TYPE INET USING ip_address::INET;

-- Revert user_sessions table
ALTER TABLE user_sessions ALTER COLUMN ip_address TYPE INET USING ip_address::INET;

