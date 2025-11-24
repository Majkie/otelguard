-- Fix ip_address column type from INET to TEXT
-- INET type can't be scanned directly into Go string type

-- Update user_feedback table
ALTER TABLE user_feedback ALTER COLUMN ip_address TYPE TEXT USING ip_address::TEXT;

-- Update user_sessions table
ALTER TABLE user_sessions ALTER COLUMN ip_address TYPE TEXT USING ip_address::TEXT;

