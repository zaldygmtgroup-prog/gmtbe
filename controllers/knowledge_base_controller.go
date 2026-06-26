package controllers

import (
	"net/http"

	"begmt2/config"
	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KnowledgeBaseController struct {
	cfg config.Config
	db  *gorm.DB
}

func NewKnowledgeBaseController(cfg config.Config, db *gorm.DB) *KnowledgeBaseController {
	return &KnowledgeBaseController{
		cfg: cfg,
		db:  db,
	}
}

// roleLabels maps role_key to a human-readable label for seeding/display purposes.
var roleLabels = map[string]string{
	"growthStrategist":        "Growth Strategist",
	"marketingSpecialist":     "Marketing Specialist",
	"conversionCommunityLead": "Conversion & Community Lead",
}

// GetAll returns all knowledge base entries as a map keyed by role_key.
// GET /api/super-admin/knowledge-base
func (kc *KnowledgeBaseController) GetAll(c *gin.Context) {
	var entries []models.ModelKnowledgeBase
	if err := kc.db.Order("role_key ASC").Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load knowledge base", "error": err.Error()})
		return
	}

	// Build a map response that matches what the frontend and content-brief-config expect.
	data := make(map[string]string, len(entries))
	for _, entry := range entries {
		data[entry.RoleKey] = entry.Content
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"entries": entries,
	})
}

type saveKnowledgeBaseRequest struct {
	GrowthStrategist        string `json:"growthStrategist"`
	MarketingSpecialist     string `json:"marketingSpecialist"`
	ConversionCommunityLead string `json:"conversionCommunityLead"`
}

// SaveAll upserts all knowledge base entries from a single request body.
// POST /api/super-admin/knowledge-base
func (kc *KnowledgeBaseController) SaveAll(c *gin.Context) {
	var req saveKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "error": err.Error()})
		return
	}

	// Get user ID from auth context for audit trail.
	userID := uint(0)
	if id, exists := c.Get("user_id"); exists {
		if uid, ok := id.(uint); ok {
			userID = uid
		}
	}

	items := map[string]string{
		"growthStrategist":        req.GrowthStrategist,
		"marketingSpecialist":     req.MarketingSpecialist,
		"conversionCommunityLead": req.ConversionCommunityLead,
	}

	tx := kc.db.Begin()
	for roleKey, content := range items {
		var entry models.ModelKnowledgeBase
		err := tx.Where("role_key = ?", roleKey).First(&entry).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				entry = models.ModelKnowledgeBase{
					RoleKey:   roleKey,
					RoleLabel: roleLabels[roleKey],
					Content:   content,
					UpdatedBy: userID,
				}
				if err := tx.Create(&entry).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save knowledge base", "error": err.Error()})
					return
				}
			} else {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"message": "database error", "error": err.Error()})
				return
			}
		} else {
			entry.Content = content
			entry.RoleLabel = roleLabels[roleKey]
			entry.UpdatedBy = userID
			if err := tx.Save(&entry).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update knowledge base", "error": err.Error()})
				return
			}
		}
	}
	tx.Commit()

	// Return the saved data in the same format as GetAll.
	data := make(map[string]string, len(items))
	for k, v := range items {
		data[k] = v
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Knowledge base saved",
		"data":    data,
	})
}

// GetByRole returns the knowledge base entry for a specific role.
// GET /api/super-admin/knowledge-base/:roleKey
func (kc *KnowledgeBaseController) GetByRole(c *gin.Context) {
	roleKey := c.Param("roleKey")

	var entry models.ModelKnowledgeBase
	err := kc.db.Where("role_key = ?", roleKey).First(&entry).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load knowledge base", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}
