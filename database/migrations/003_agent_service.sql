CREATE TABLE IF NOT EXISTS agent_wallets (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  total_commission BIGINT NOT NULL DEFAULT 0,
  available_balance BIGINT NOT NULL DEFAULT 0,
  pending_withdraw BIGINT NOT NULL DEFAULT 0,
  withdrawn_balance BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_agent_wallets_user_id (user_id),
  CONSTRAINT fk_agent_wallets_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS agent_commissions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  product_name VARCHAR(150) NOT NULL,
  product_price BIGINT NOT NULL,
  discount_amount BIGINT NOT NULL DEFAULT 0,
  final_price BIGINT NOT NULL,
  commission_percent DOUBLE NOT NULL,
  commission_amount BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  KEY idx_agent_commissions_user_id (user_id),
  CONSTRAINT fk_agent_commissions_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS withdraw_requests (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  amount BIGINT NOT NULL,
  status ENUM('on_progress','approval') NOT NULL DEFAULT 'on_progress',
  approved_at DATETIME(3) NULL,
  approved_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  KEY idx_withdraw_requests_user_id (user_id),
  CONSTRAINT fk_withdraw_requests_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
);
