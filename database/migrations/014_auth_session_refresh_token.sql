ALTER TABLE auth_sessions
  ADD COLUMN refresh_token_hash VARCHAR(64) NULL AFTER client,
  ADD KEY idx_auth_sessions_refresh_token_hash (refresh_token_hash);
