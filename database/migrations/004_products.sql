CREATE TABLE IF NOT EXISTS products (
    id_product BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    namaproduct VARCHAR(150) NOT NULL,
    foto VARCHAR(255) NULL,
    deskripsi TEXT NULL,
    unit VARCHAR(50) NOT NULL,
    price BIGINT NOT NULL,
    status VARCHAR(50) NULL,
    komisi DOUBLE NULL DEFAULT 0,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL
);

CREATE INDEX idx_products_deleted_at ON products (deleted_at);

-- Index untuk pencarian namaproduct dan deskripsi
CREATE INDEX idx_products_search ON products (namaproduct);