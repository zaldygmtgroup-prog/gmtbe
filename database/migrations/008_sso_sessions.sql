CREATE TABLE IF NOT EXISTS auth_sessions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  session_id VARCHAR(191) NOT NULL UNIQUE,
  user_id BIGINT UNSIGNED NOT NULL,
  client VARCHAR(100) NULL,
  expires_at DATETIME(3) NOT NULL,
  revoked_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  KEY idx_auth_sessions_user_id (user_id),
  CONSTRAINT fk_auth_sessions_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sso_codes (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  code_hash VARCHAR(255) NOT NULL UNIQUE,
  user_id BIGINT UNSIGNED NOT NULL,
  target_client VARCHAR(100) NOT NULL,
  redirect_uri VARCHAR(255) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  used_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  KEY idx_sso_codes_user_id (user_id),
  KEY idx_sso_codes_target_client (target_client),
  CONSTRAINT fk_sso_codes_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
);
