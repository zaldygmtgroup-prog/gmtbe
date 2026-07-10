-- Migration: articles table for CMS article management and scraping import
-- Depends on: none

CREATE TABLE IF NOT EXISTS articles (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    title          VARCHAR(500)    NOT NULL,
    slug           VARCHAR(500)    NOT NULL,
    excerpt        TEXT,
    content        LONGTEXT,
    featured_image VARCHAR(500),
    author         VARCHAR(255),
    source_url     VARCHAR(500),
    status         VARCHAR(50)     NOT NULL DEFAULT 'draft',
    seo            JSON,
    published_at   DATETIME(3),
    created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at     DATETIME(3),
    UNIQUE INDEX idx_articles_slug (slug),
    INDEX idx_articles_status (status),
    INDEX idx_articles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
