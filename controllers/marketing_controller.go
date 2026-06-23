package controllers

import (
	"net/http"
	"time"

	"begmt2/config"
	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MarketingController struct {
	cfg config.Config
	db  *gorm.DB
}

func NewMarketingController(cfg config.Config, db *gorm.DB) *MarketingController {
	return &MarketingController{
		cfg: cfg,
		db:  db,
	}
}

type saveCacheRequest struct {
	IGUserID          string                    `json:"ig_user_id" binding:"required"`
	IGUsername        string                    `json:"ig_username"`
	ContentBrief      models.ContentBrief       `json:"content_brief"`
	ContentReasoning  []models.ContentReasoning `json:"content_reasoning"`
	ContentReferences []models.ContentReference `json:"content_references"`
}

func (mc *MarketingController) GetCache(c *gin.Context) {
	igUserID := c.Query("ig_user_id")
	if igUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ig_user_id query parameter is required"})
		return
	}

	var cache models.AIContentBriefCache
	err := mc.db.Where("ig_user_id = ?", igUserID).First(&cache).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"cached": false,
				"data":   nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to query cache", "error": err.Error()})
		return
	}

	// Check if expired
	if time.Now().After(cache.ExpiresAt) {
		c.JSON(http.StatusOK, gin.H{
			"cached": false,
			"data":   nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cached": true,
		"data":   cache,
	})
}

func (mc *MarketingController) SaveCache(c *gin.Context) {
	var req saveCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "error": err.Error()})
		return
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, 7) // 7 days later

	var cache models.AIContentBriefCache
	err := mc.db.Where("ig_user_id = ?", req.IGUserID).First(&cache).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			cache = models.AIContentBriefCache{
				IGUserID:          req.IGUserID,
				IGUsername:        req.IGUsername,
				ContentBrief:      models.JSONField[models.ContentBrief]{Val: req.ContentBrief},
				ContentReasoning:  models.JSONField[[]models.ContentReasoning]{Val: req.ContentReasoning},
				ContentReferences: models.JSONField[[]models.ContentReference]{Val: req.ContentReferences},
				GeneratedAt:       now,
				ExpiresAt:         expiresAt,
			}
			if err := mc.db.Create(&cache).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save content brief cache", "error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "database error", "error": err.Error()})
			return
		}
	} else {
		// Update and reset generated_at and expires_at
		cache.IGUsername = req.IGUsername
		cache.ContentBrief = models.JSONField[models.ContentBrief]{Val: req.ContentBrief}
		cache.ContentReasoning = models.JSONField[[]models.ContentReasoning]{Val: req.ContentReasoning}
		cache.ContentReferences = models.JSONField[[]models.ContentReference]{Val: req.ContentReferences}
		cache.GeneratedAt = now
		cache.ExpiresAt = expiresAt
		if err := mc.db.Save(&cache).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update content brief cache", "error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Content brief cache saved",
		"data": gin.H{
			"id":           cache.ID,
			"ig_user_id":   cache.IGUserID,
			"ig_username":  cache.IGUsername,
			"generated_at": cache.GeneratedAt,
			"expires_at":   cache.ExpiresAt,
		},
	})
}

func (mc *MarketingController) DeleteCache(c *gin.Context) {
	igUserID := c.Query("ig_user_id")
	if igUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ig_user_id query parameter is required"})
		return
	}

	err := mc.db.Where("ig_user_id = ?", igUserID).Delete(&models.AIContentBriefCache{}).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete content brief cache", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Content brief cache deleted",
	})
}
