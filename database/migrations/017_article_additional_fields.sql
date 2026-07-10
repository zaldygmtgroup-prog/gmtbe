-- Migration: Add category and metadata to articles
-- Depends on: 016_articles.sql

ALTER TABLE articles 
ADD COLUMN category VARCHAR(255) AFTER slug,
ADD COLUMN metadata JSON AFTER seo;
