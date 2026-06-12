CREATE TABLE IF NOT EXISTS products (
  id_product BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  namaproduct VARCHAR(150) NOT NULL,
  foto VARCHAR(255) NULL,
  deskripsi TEXT NULL,
  unit VARCHAR(50) NOT NULL,
  price BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL
);

CREATE INDEX idx_products_search ON products (namaproduct);
