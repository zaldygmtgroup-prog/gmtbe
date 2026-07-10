DROP TABLE IF EXISTS preorder_items;
DROP TABLE IF EXISTS detail_preorders;
DROP TABLE IF EXISTS preorders;

CREATE TABLE preorders (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  po_number VARCHAR(50) NOT NULL UNIQUE,
  id_agent BIGINT UNSIGNED NOT NULL,
  nama_customer VARCHAR(255) NOT NULL,
  nama_perusahaan VARCHAR(255) NULL,
  email VARCHAR(255) NOT NULL,
  alamat TEXT NOT NULL,
  no_hp VARCHAR(50) NOT NULL,
  catatan TEXT NULL,
  subtotal BIGINT NOT NULL,
  total_discount BIGINT NOT NULL DEFAULT 0,
  total BIGINT NOT NULL,
  total_komisi BIGINT NOT NULL DEFAULT 0,
  status ENUM('draft','in_review','approve','invalid') NOT NULL DEFAULT 'draft',
  invalid_reason TEXT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  CONSTRAINT fk_preorders_agent FOREIGN KEY (id_agent) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE preorder_items (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  id_preorder BIGINT UNSIGNED NOT NULL,
  id_product BIGINT UNSIGNED NOT NULL,
  product_name_snapshot VARCHAR(255) NOT NULL,
  product_photo_snapshot VARCHAR(255) NULL,
  product_description_snapshot TEXT NULL,
  unit_snapshot VARCHAR(50) NULL,
  unit_price BIGINT NOT NULL,
  qty INT NOT NULL,
  discount_percent DOUBLE NOT NULL DEFAULT 0,
  discount_amount BIGINT NOT NULL DEFAULT 0,
  subtotal BIGINT NOT NULL,
  total BIGINT NOT NULL,
  komisi BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  CONSTRAINT fk_preorder_items_preorder FOREIGN KEY (id_preorder) REFERENCES preorders(id) ON DELETE CASCADE,
  CONSTRAINT fk_preorder_items_product FOREIGN KEY (id_product) REFERENCES products(id_product) ON DELETE RESTRICT
);

DROP TABLE IF EXISTS withdraw_requests;
CREATE TABLE withdraw_requests (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  withdraw_number VARCHAR(50) NOT NULL UNIQUE,
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
