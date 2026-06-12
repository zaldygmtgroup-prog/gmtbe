CREATE TABLE IF NOT EXISTS preorders (
  id_preorder BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  id_product BIGINT UNSIGNED NOT NULL,
  id_agent BIGINT UNSIGNED NOT NULL,
  qty BIGINT NOT NULL,
  subtotal BIGINT NOT NULL,
  total_komisi BIGINT NOT NULL,
  total BIGINT NOT NULL,
  status ENUM('draft','in_review','approve','invalid') NOT NULL DEFAULT 'draft',
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  KEY idx_preorders_id_product (id_product),
  KEY idx_preorders_id_agent (id_agent),
  KEY idx_preorders_status (status),
  CONSTRAINT fk_preorders_product
    FOREIGN KEY (id_product) REFERENCES products(id_product)
    ON DELETE RESTRICT,
  CONSTRAINT fk_preorders_agent
    FOREIGN KEY (id_agent) REFERENCES users(id)
    ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS detail_preorders (
  id_detail_preorder BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  id_preorder BIGINT UNSIGNED NOT NULL,
  nama_customer VARCHAR(150) NULL,
  email VARCHAR(191) NULL,
  alamat TEXT NULL,
  no_hp VARCHAR(30) NULL,
  catatan TEXT NULL,
  reviewed_by BIGINT UNSIGNED NULL,
  reviewed_at DATETIME(3) NULL,
  invalid_reason TEXT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_detail_preorders_id_preorder (id_preorder),
  CONSTRAINT fk_detail_preorders_preorder
    FOREIGN KEY (id_preorder) REFERENCES preorders(id_preorder)
    ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  role VARCHAR(30) NOT NULL,
  title VARCHAR(150) NOT NULL,
  message TEXT NOT NULL,
  data JSON NULL,
  read_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  KEY idx_notifications_role (role)
);
