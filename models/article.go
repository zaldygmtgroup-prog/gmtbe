package models

import (
	"time"

	"gorm.io/gorm"
)

// ArticleSEO holds per-article SEO metadata, stored as a JSON column.
type ArticleSEO struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	CanonicalURL string `json:"canonical_url"`
}

// ArticleMetadata holds arrays of linked media and related items.
type ArticleMetadata struct {
	Gallery         []string `json:"gallery"`
	RelatedProducts []uint   `json:"related_products"`
	RelatedArticles []uint   `json:"related_articles"`
}

// Article represents a CMS article, compatible with the scraping migration script.
type Article struct {
	ID            uint                  `gorm:"primaryKey;column:id"             json:"id"`
	Title         string                  `gorm:"size:500;not null;column:title"   json:"title"`
	Slug          string                  `gorm:"size:500;not null;uniqueIndex;column:slug" json:"slug"`
	Category      string                  `gorm:"size:255;column:category"         json:"category"`
	Excerpt       string                  `gorm:"type:text;column:excerpt"         json:"excerpt"`
	Content       string                  `gorm:"type:longtext;column:content"     json:"content"`
	FeaturedImage string                  `gorm:"size:500;column:featured_image"   json:"featured_image"`
	Author        string                  `gorm:"size:255;column:author"           json:"author"`
	SourceURL     string                  `gorm:"size:500;column:source_url"       json:"source_url"`
	Status        string                  `gorm:"size:50;not null;default:draft;column:status" json:"status"`
	SEO           JSONField[ArticleSEO]   `gorm:"type:json;column:seo"             json:"seo"`
	Metadata      JSONField[ArticleMetadata] `gorm:"type:json;column:metadata"     json:"metadata"`
	PublishedAt   *time.Time            `gorm:"column:published_at"              json:"published_at"`
	CreatedAt     time.Time             `gorm:"column:created_at"                json:"created_at"`
	UpdatedAt     time.Time             `gorm:"column:updated_at"                json:"updated_at"`
	DeletedAt     gorm.DeletedAt        `gorm:"index;column:deleted_at"          json:"-"`
}
