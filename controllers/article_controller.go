package controllers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"begmt2/config"
	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ArticleController handles CMS article CRUD and bulk import.
type ArticleController struct {
	cfg config.Config
	db  *gorm.DB
}

// NewArticleController creates a new ArticleController.
func NewArticleController(cfg config.Config, db *gorm.DB) *ArticleController {
	return &ArticleController{cfg: cfg, db: db}
}

// ---------------------------------------------------------------------------
// Request / response structs
// ---------------------------------------------------------------------------

type articleRequest struct {
	Title         string              `json:"title"          binding:"required,max=500"`
	Slug          string                  `json:"slug"           binding:"required,max=500"`
	Category      string                  `json:"category"       binding:"omitempty,max=255"`
	Excerpt       string                  `json:"excerpt"`
	Content       string                  `json:"content"`
	FeaturedImage string                  `json:"featured_image" binding:"omitempty,max=500"`
	Author        string                  `json:"author"         binding:"omitempty,max=255"`
	SourceURL     string                  `json:"source_url"     binding:"omitempty,max=500"`
	Status        string                  `json:"status"         binding:"omitempty,oneof=draft published archived"`
	SEO           *models.ArticleSEO      `json:"seo"`
	Metadata      *models.ArticleMetadata `json:"metadata"`
	PublishedAt   *time.Time              `json:"published_at"`
	UpdatedAt     *time.Time              `json:"updated_at"`
}

type articleSummary struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

type bulkImportRequest struct {
	Articles []articleRequest `json:"articles" binding:"required,min=1,dive"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseArticleID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid article id"})
		return 0, false
	}
	return uint(id), true
}

func applyArticleRequest(a *models.Article, req *articleRequest) {
	a.Title = req.Title
	a.Slug = req.Slug
	a.Category = req.Category
	a.Excerpt = req.Excerpt
	a.Content = req.Content
	a.FeaturedImage = req.FeaturedImage
	a.Author = req.Author
	a.SourceURL = req.SourceURL

	if req.Status != "" {
		a.Status = req.Status
	} else if a.Status == "" {
		a.Status = "draft"
	}

	if req.SEO != nil {
		a.SEO = models.JSONField[models.ArticleSEO]{Val: *req.SEO}
	}

	if req.Metadata != nil {
		a.Metadata = models.JSONField[models.ArticleMetadata]{Val: *req.Metadata}
	}

	if req.PublishedAt != nil {
		a.PublishedAt = req.PublishedAt
	}

	// Allow the scraper to set its own updated_at.
	if req.UpdatedAt != nil {
		a.UpdatedAt = *req.UpdatedAt
	}
}

// ---------------------------------------------------------------------------
// GET /api/articles  — list with search, status filter, and pagination
// ---------------------------------------------------------------------------

func (ac *ArticleController) ListArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 {
		limit = 20
	}

	query := ac.db.Model(&models.Article{})

	if search := c.Query("search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("title LIKE ? OR excerpt LIKE ?", like, like)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var articles []models.Article
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get articles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"articles": articles,
		"meta": gin.H{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

// ---------------------------------------------------------------------------
// GET /api/articles/:id  — single article by ID or slug
// ---------------------------------------------------------------------------

func (ac *ArticleController) GetArticle(c *gin.Context) {
	param := c.Param("id")

	var article models.Article
	var err error

	// If param is numeric, look up by ID; otherwise treat as slug.
	if id, parseErr := strconv.ParseUint(param, 10, 64); parseErr == nil {
		err = ac.db.First(&article, "id = ?", id).Error
	} else {
		err = ac.db.Where("slug = ?", param).First(&article).Error
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"article": article})
}

// ---------------------------------------------------------------------------
// POST /api/articles  — create single article
// ---------------------------------------------------------------------------

func (ac *ArticleController) CreateArticle(c *gin.Context) {
	var req articleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	// Ensure slug uniqueness.
	var count int64
	ac.db.Model(&models.Article{}).Where("slug = ?", req.Slug).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "slug already exists"})
		return
	}

	var article models.Article
	applyArticleRequest(&article, &req)

	if err := ac.db.Create(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create article"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Article created",
		"article": articleSummary{
			ID:     article.ID,
			Title:  article.Title,
			Slug:   article.Slug,
			Status: article.Status,
		},
	})
}

// ---------------------------------------------------------------------------
// PUT /api/articles/:id  — update article
// ---------------------------------------------------------------------------

func (ac *ArticleController) UpdateArticle(c *gin.Context) {
	articleID, ok := parseArticleID(c)
	if !ok {
		return
	}

	var article models.Article
	if err := ac.db.First(&article, "id = ?", articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get article"})
		return
	}

	var req articleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	// If slug changed, check uniqueness.
	if req.Slug != article.Slug {
		var count int64
		ac.db.Model(&models.Article{}).Where("slug = ? AND id != ?", req.Slug, articleID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"message": "slug already exists"})
			return
		}
	}

	applyArticleRequest(&article, &req)

	if err := ac.db.Save(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated", "article": article})
}

// ---------------------------------------------------------------------------
// DELETE /api/articles/:id  — soft-delete article
// ---------------------------------------------------------------------------

func (ac *ArticleController) DeleteArticle(c *gin.Context) {
	articleID, ok := parseArticleID(c)
	if !ok {
		return
	}

	result := ac.db.Delete(&models.Article{}, "id = ?", articleID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete article"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "article not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article deleted"})
}

// ---------------------------------------------------------------------------
// POST /api/articles/import  — bulk import articles
// ---------------------------------------------------------------------------

func (ac *ArticleController) ImportArticles(c *gin.Context) {
	var req bulkImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	created := make([]articleSummary, 0, len(req.Articles))
	skipped := make([]string, 0)

	err := ac.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Articles {
			// Skip duplicates by slug.
			var count int64
			tx.Model(&models.Article{}).Where("slug = ?", item.Slug).Count(&count)
			if count > 0 {
				skipped = append(skipped, item.Slug)
				continue
			}

			var article models.Article
			applyArticleRequest(&article, &item)

			if err := tx.Create(&article).Error; err != nil {
				// If it's a duplicate key error, skip gracefully.
				if strings.Contains(err.Error(), "Duplicate") {
					skipped = append(skipped, item.Slug)
					continue
				}
				return err
			}

			created = append(created, articleSummary{
				ID:     article.ID,
				Title:  article.Title,
				Slug:   article.Slug,
				Status: article.Status,
			})
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to import articles", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Import completed",
		"created_count": len(created),
		"skipped_count": len(skipped),
		"created":       created,
		"skipped_slugs": skipped,
	})
}
