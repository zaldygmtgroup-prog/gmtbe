package controllers

import (
	"errors"
	"net/http"
	"time"

	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AgentOnboardingController struct {
	db *gorm.DB
}

func NewAgentOnboardingController(db *gorm.DB) AgentOnboardingController {
	return AgentOnboardingController{db: db}
}

// GET /api/agent/onboarding/videos
func (ctrl AgentOnboardingController) ListVideos(c *gin.Context) {
	var videos []models.AgentOnboardingVideo
	if err := ctrl.db.Order("sort_order ASC, id ASC").Find(&videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to fetch videos", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"videos": videos})
}

// GET /api/agent/onboarding/progress
func (ctrl AgentOnboardingController) GetProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	var videos []models.AgentOnboardingVideo
	if err := ctrl.db.Order("sort_order ASC, id ASC").Find(&videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to fetch videos", "error": err.Error()})
		return
	}

	var progressList []models.AgentOnboardingProgress
	if err := ctrl.db.Where("user_id = ?", userID).Find(&progressList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to fetch progress", "error": err.Error()})
		return
	}

	progressMap := make(map[uint]models.AgentOnboardingProgress)
	for _, p := range progressList {
		progressMap[p.VideoID] = p
	}

	var totalRequired int
	var completedCount int
	var progressResponse []gin.H

	for _, v := range videos {
		if v.IsRequired {
			totalRequired++
		}

		p, exists := progressMap[v.ID]
		if exists {
			if v.IsRequired && p.Status == models.OnboardingStatusCompleted {
				completedCount++
			}

			progressResponse = append(progressResponse, gin.H{
				"video_id":        v.ID,
				"slug":            v.Slug,
				"status":          p.Status,
				"watched_seconds": p.WatchedSeconds,
				"completed_at":    p.CompletedAt,
			})
		} else {
			progressResponse = append(progressResponse, gin.H{
				"video_id":        v.ID,
				"slug":            v.Slug,
				"status":          models.OnboardingStatusNotStarted,
				"watched_seconds": 0,
				"completed_at":    nil,
			})
		}
	}

	completionPercent := 0
	if totalRequired > 0 {
		completionPercent = (completedCount * 100) / totalRequired
	} else if len(videos) == 0 {
		completionPercent = 100
	} else {
		completionPercent = 100
	}

	isCompleted := false
	if totalRequired > 0 {
		isCompleted = completedCount == totalRequired
	} else {
		isCompleted = true
	}

	c.JSON(http.StatusOK, gin.H{
		"completed_count":    completedCount,
		"total_required":     totalRequired,
		"completion_percent": completionPercent,
		"is_completed":       isCompleted,
		"progress":           progressResponse,
	})
}

type SaveProgressRequest struct {
	VideoID         uint                    `json:"video_id" binding:"required"`
	WatchedSeconds  int                     `json:"watched_seconds" binding:"min=0"`
	DurationSeconds int                     `json:"duration_seconds" binding:"required,min=1"`
	Status          models.OnboardingStatus `json:"status" binding:"required,oneof=not_started in_progress completed"`
}

// POST /api/agent/onboarding/progress
func (ctrl AgentOnboardingController) SaveProgress(c *gin.Context) {
	var req SaveProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body", "error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	var savedProgress models.AgentOnboardingProgress

	err := ctrl.db.Transaction(func(tx *gorm.DB) error {
		var video models.AgentOnboardingVideo
		if err := tx.First(&video, req.VideoID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("video not found")
			}
			return err
		}

		// Check if progress record already exists
		var progress models.AgentOnboardingProgress
		hasExisting := true
		err := tx.Where("user_id = ? AND video_id = ?", userID, req.VideoID).First(&progress).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				hasExisting = false
			} else {
				return err
			}
		}

		// Determine status
		status := req.Status
		if float64(req.WatchedSeconds) >= float64(video.DurationSeconds)*0.9 {
			status = models.OnboardingStatusCompleted
		}

		// If it was already completed, keep it completed
		if hasExisting && progress.Status == models.OnboardingStatusCompleted {
			status = models.OnboardingStatusCompleted
		}

		// Validation rules:
		// 1. If status is completed, watched_seconds must be >= duration_seconds * 0.9
		if status == models.OnboardingStatusCompleted && float64(req.WatchedSeconds) < float64(video.DurationSeconds)*0.9 {
			return errors.New("cannot complete video: watched seconds must be at least 90% of video duration")
		}

		// 2. video 2 tidak bisa completed kalau video 1 belum completed (validate order)
		if status == models.OnboardingStatusCompleted {
			var allVideos []models.AgentOnboardingVideo
			if err := tx.Order("sort_order ASC, id ASC").Find(&allVideos).Error; err != nil {
				return err
			}

			var precedingVideoIDs []uint
			for _, v := range allVideos {
				// Videos with smaller sort order, or same sort order but smaller ID, are preceding
				if v.SortOrder < video.SortOrder || (v.SortOrder == video.SortOrder && v.ID < video.ID) {
					precedingVideoIDs = append(precedingVideoIDs, v.ID)
				}
			}

			if len(precedingVideoIDs) > 0 {
				var completedCount int64
				if err := tx.Model(&models.AgentOnboardingProgress{}).
					Where("user_id = ? AND video_id IN ? AND status = ?", userID, precedingVideoIDs, models.OnboardingStatusCompleted).
					Count(&completedCount).Error; err != nil {
					return err
				}

				if int(completedCount) < len(precedingVideoIDs) {
					return errors.New("cannot complete video: preceding videos must be completed first")
				}
			}
		}

		now := time.Now()
		if !hasExisting {
			progress = models.AgentOnboardingProgress{
				UserID:         userID,
				VideoID:        req.VideoID,
				Status:         status,
				WatchedSeconds: req.WatchedSeconds,
			}
			if status == models.OnboardingStatusCompleted {
				progress.CompletedAt = &now
			}
			if err := tx.Create(&progress).Error; err != nil {
				return err
			}
		} else {
			var completedAt *time.Time
			if status == models.OnboardingStatusCompleted {
				if progress.CompletedAt != nil {
					completedAt = progress.CompletedAt
				} else {
					completedAt = &now
				}
			}

			updates := map[string]interface{}{
				"status":          status,
				"watched_seconds": req.WatchedSeconds,
				"completed_at":    completedAt,
			}
			if err := tx.Model(&progress).Updates(updates).Error; err != nil {
				return err
			}
			progress.Status = status
			progress.WatchedSeconds = req.WatchedSeconds
			progress.CompletedAt = completedAt
		}

		savedProgress = progress
		return nil
	})

	if err != nil {
		if err.Error() == "video not found" {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		if err.Error() == "cannot complete video: watched seconds must be at least 90% of video duration" ||
			err.Error() == "cannot complete video: preceding videos must be completed first" {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save progress", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Progress saved",
		"progress": gin.H{
			"video_id":        savedProgress.VideoID,
			"status":          savedProgress.Status,
			"watched_seconds": savedProgress.WatchedSeconds,
			"completed_at":    savedProgress.CompletedAt,
		},
	})
}

// DELETE /api/agent/onboarding/progress
func (ctrl AgentOnboardingController) ResetProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	if err := ctrl.db.Where("user_id = ?", userID).Delete(&models.AgentOnboardingProgress{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to reset progress", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Progress reset successfully"})
}
