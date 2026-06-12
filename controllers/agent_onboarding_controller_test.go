package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"begmt2/config"
	"begmt2/controllers"
	"begmt2/models"
	"begmt2/seeders"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	cfg := config.Load()
	db := config.ConnectDatabase(cfg)

	err := db.AutoMigrate(
		&models.User{},
		&models.AgentOnboardingVideo{},
		&models.AgentOnboardingProgress{},
		&models.Product{},
		&models.Preorder{},
		&models.PreorderItem{},
		&models.AgentWallet{},
		&models.AgentCommission{},
		&models.WithdrawRequest{},
		&models.Notification{},
		&models.AuthSession{},
		&models.SSOCode{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestAgentOnboarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	// Run within transaction so we don't pollute the database
	tx := db.Begin()
	defer tx.Rollback()

	// Seed videos
	seeders.SeedOnboardingVideos(tx)

	// Create test agent user
	agentUser := models.User{
		Name:  "Test Agent",
		Email: "testagent@example.com",
		Role:  models.RoleAgent,
	}
	if err := tx.Create(&agentUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	ctrl := controllers.NewAgentOnboardingController(tx)

	// 1. Test ListVideos
	t.Run("ListVideos", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/agent/onboarding/videos", ctrl.ListVideos)

		req, _ := http.NewRequest("GET", "/api/agent/onboarding/videos", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string][]models.AgentOnboardingVideo
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp["videos"]) != 3 {
			t.Errorf("expected 3 videos, got %d", len(resp["videos"]))
		}
	})

	// 2. Test GetProgress (Initially all not_started)
	t.Run("GetProgress - Initial", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", agentUser.ID)
			c.Next()
		})
		r.GET("/api/agent/onboarding/progress", ctrl.GetProgress)

		req, _ := http.NewRequest("GET", "/api/agent/onboarding/progress", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if resp["completed_count"].(float64) != 0 {
			t.Errorf("expected completed_count 0, got %v", resp["completed_count"])
		}
		if resp["is_completed"].(bool) != false {
			t.Errorf("expected is_completed false")
		}
	})

	// 3. Test SaveProgress for video 2 before video 1 is completed (should fail)
	t.Run("SaveProgress - Out of order completion", func(t *testing.T) {
		var videos []models.AgentOnboardingVideo
		tx.Order("sort_order ASC").Find(&videos)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", agentUser.ID)
			c.Next()
		})
		r.POST("/api/agent/onboarding/progress", ctrl.SaveProgress)

		body := map[string]interface{}{
			"video_id":         videos[1].ID,
			"watched_seconds":  videos[1].DurationSeconds,
			"duration_seconds": videos[1].DurationSeconds,
			"status":           "completed",
		}
		b, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/agent/onboarding/progress", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 4. Test SaveProgress for video 1 (should succeed and auto-complete)
	t.Run("SaveProgress - Valid video 1 progress", func(t *testing.T) {
		var videos []models.AgentOnboardingVideo
		tx.Order("sort_order ASC").Find(&videos)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", agentUser.ID)
			c.Next()
		})
		r.POST("/api/agent/onboarding/progress", ctrl.SaveProgress)

		watched := int(float64(videos[0].DurationSeconds) * 0.95)
		body := map[string]interface{}{
			"video_id":         videos[0].ID,
			"watched_seconds":  watched,
			"duration_seconds": videos[0].DurationSeconds,
			"status":           "in_progress",
		}
		b, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/agent/onboarding/progress", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		progressMap := resp["progress"].(map[string]interface{})
		if progressMap["status"].(string) != "completed" {
			t.Errorf("expected progress status to be promoted to completed, got %s", progressMap["status"])
		}
	})
}
