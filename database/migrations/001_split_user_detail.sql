CREATE TABLE IF NOT EXISTS detail_users (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  company_name VARCHAR(150) NOT NULL,
  job VARCHAR(120) NULL,
  instagram VARCHAR(120) NULL,
  facebook VARCHAR(120) NULL,
  tiktok VARCHAR(120) NULL,
  photo VARCHAR(255) NULL,
  ktp_photo VARCHAR(255) NULL,
  full_address TEXT NULL,
  bank_name VARCHAR(120) NULL,
  account_number VARCHAR(80) NULL,
  status VARCHAR(50) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_detail_users_user_id (user_id),
  CONSTRAINT fk_detail_users_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
);

INSERT INTO detail_users (
  user_id,
  company_name,
  job,
  instagram,
  facebook,
  tiktok,
  created_at,
  updated_at
)
SELECT
  id,
  company_name,
  job,
  instagram,
  facebook,
  tiktok,
  NOW(3),
  NOW(3)
FROM users
LEFT JOIN detail_users ON detail_users.user_id = users.id
WHERE company_name IS NOT NULL
  AND company_name <> ''
  AND detail_users.id IS NULL;

ALTER TABLE users
  DROP COLUMN company_name,
  DROP COLUMN job,
  DROP COLUMN instagram,
  DROP COLUMN facebook,
  DROP COLUMN tiktok;
