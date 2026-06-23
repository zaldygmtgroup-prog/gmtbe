package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"begmt2/config"
	"begmt2/controllers"
	"begmt2/middleware"
	"begmt2/models"
	"begmt2/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func generateTestToken(t *testing.T, tx *gorm.DB, user models.User, cfg config.Config) string {
	sessionID := "session-" + user.Email
	session := models.AuthSession{
		SessionID: sessionID,
		UserID:    user.ID,
		Client:    "test",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := tx.Create(&session).Error; err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, string(user.Role), sessionID, cfg.JWTSecret, cfg.JWTExpiresHours)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}
	return token
}

func TestMarketingController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	tx := db.Begin()
	defer tx.Rollback()

	// Seed marketing user and superadmin user
	marketingUser := models.User{
		Name:        "Marketing User",
		Email:       "marketing@example.com",
		Role:        models.RoleMarketing,
		PhoneNumber: "123",
		Gender:      "M",
		TTL:         "Jakarta, 1990",
		Domicile:    "Jakarta",
	}
	if err := tx.Create(&marketingUser).Error; err != nil {
		t.Fatalf("failed to create marketing user: %v", err)
	}

	superAdminUser := models.User{
		Name:        "Super Admin User",
		Email:       "superadmin@example.com",
		Role:        models.RoleSuperAdmin,
		PhoneNumber: "456",
		Gender:      "F",
		TTL:         "Jakarta, 1991",
		Domicile:    "Jakarta",
	}
	if err := tx.Create(&superAdminUser).Error; err != nil {
		t.Fatalf("failed to create super admin user: %v", err)
	}

	regularUser := models.User{
		Name:        "Regular User",
		Email:       "user@example.com",
		Role:        models.RoleUser,
		PhoneNumber: "789",
		Gender:      "M",
		TTL:         "Jakarta, 1992",
		Domicile:    "Jakarta",
	}
	if err := tx.Create(&regularUser).Error; err != nil {
		t.Fatalf("failed to create regular user: %v", err)
	}

	cfg := config.Config{
		JWTSecret:       "test-secret-key-12345",
		JWTExpiresHours: 24,
	}

	marketingToken := generateTestToken(t, tx, marketingUser, cfg)
	superAdminToken := generateTestToken(t, tx, superAdminUser, cfg)
	regularToken := generateTestToken(t, tx, regularUser, cfg)

	ctrl := controllers.NewMarketingController(cfg, tx)

	r := gin.New()
	marketingGroup := r.Group("/api/marketing")
	{
		marketingGroup.GET("/content-brief-cache", ctrl.GetCache)
		marketingGroup.POST("/content-brief-cache", middleware.AuthMiddleware(cfg, tx), middleware.RoleMiddleware("marketing", "super_admin"), ctrl.SaveCache)
		marketingGroup.DELETE("/content-brief-cache", middleware.AuthMiddleware(cfg, tx), middleware.RoleMiddleware("marketing", "super_admin"), ctrl.DeleteCache)
	}

	t.Run("GET Cache Miss", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/marketing/content-brief-cache?ig_user_id=123", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["cached"].(bool) != false {
			t.Errorf("expected cached false")
		}
		if resp["data"] != nil {
			t.Errorf("expected data nil")
		}
	})

	t.Run("GET Missing Query Param", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/marketing/content-brief-cache", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("POST Save Cache - Unauthorized", func(t *testing.T) {
		body := map[string]interface{}{
			"ig_user_id":  "123",
			"ig_username": "test.user",
		}
		w := performJSONRequest(t, r, "POST", "/api/marketing/content-brief-cache", body, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("POST Save Cache - Forbidden (regular user)", func(t *testing.T) {
		body := map[string]interface{}{
			"ig_user_id":  "123",
			"ig_username": "test.user",
		}
		w := performJSONRequest(t, r, "POST", "/api/marketing/content-brief-cache", body, regularToken)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})

	t.Run("POST Save Cache - Success (marketing role)", func(t *testing.T) {
		body := map[string]interface{}{
			"ig_user_id":  "123",
			"ig_username": "test.user",
			"content_brief": map[string]interface{}{
				"source":  "alibaba",
				"summary": "Fokus ke Reels...",
				"items": []map[string]interface{}{
					{
						"day":    "Hari 1",
						"format": "Reels",
						"idea":   "Idea 1",
					},
				},
			},
			"content_reasoning": []map[string]interface{}{
				{
					"media_id":  "media_1",
					"reasoning": "Reason 1",
				},
			},
			"content_references": []map[string]interface{}{
				{
					"id":          "ref_1",
					"title":       "Title 1",
					"contentType": "Reels",
				},
			},
		}
		w := performJSONRequest(t, r, "POST", "/api/marketing/content-brief-cache", body, marketingToken)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["message"].(string) != "Content brief cache saved" {
			t.Errorf("expected message 'Content brief cache saved'")
		}
		data := resp["data"].(map[string]interface{})
		if data["ig_user_id"].(string) != "123" {
			t.Errorf("expected ig_user_id '123'")
		}
		if data["ig_username"].(string) != "test.user" {
			t.Errorf("expected ig_username 'test.user'")
		}
		if data["generated_at"] == "" || data["expires_at"] == "" {
			t.Errorf("expected generated_at and expires_at to be set")
		}
	})

	t.Run("GET Cache Hit", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/marketing/content-brief-cache?ig_user_id=123", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["cached"].(bool) != true {
			t.Errorf("expected cached true")
		}
		data := resp["data"].(map[string]interface{})
		if data["ig_user_id"].(string) != "123" {
			t.Errorf("expected ig_user_id '123'")
		}
		
		// Verify structures
		brief := data["content_brief"].(map[string]interface{})
		if brief["source"].(string) != "alibaba" {
			t.Errorf("expected brief source 'alibaba'")
		}
		items := brief["items"].([]interface{})
		if len(items) != 1 {
			t.Errorf("expected 1 brief item")
		}
		
		reasoning := data["content_reasoning"].([]interface{})
		if len(reasoning) != 1 {
			t.Errorf("expected 1 reasoning item")
		}
		
		refs := data["content_references"].([]interface{})
		if len(refs) != 1 {
			t.Errorf("expected 1 reference item")
		}
	})

	t.Run("POST Save Cache - Update/Upsert (super_admin role)", func(t *testing.T) {
		body := map[string]interface{}{
			"ig_user_id":  "123",
			"ig_username": "updated.user",
			"content_brief": map[string]interface{}{
				"source":  "updated-source",
				"summary": "Updated summary",
				"items":   []map[string]interface{}{},
			},
			"content_reasoning":  []map[string]interface{}{},
			"content_references": []map[string]interface{}{},
		}
		w := performJSONRequest(t, r, "POST", "/api/marketing/content-brief-cache", body, superAdminToken)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Query again
		wGet := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/marketing/content-brief-cache?ig_user_id=123", nil)
		r.ServeHTTP(wGet, req)

		var resp map[string]interface{}
		json.Unmarshal(wGet.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		if data["ig_username"].(string) != "updated.user" {
			t.Errorf("expected updated username 'updated.user'")
		}
		brief := data["content_brief"].(map[string]interface{})
		if brief["source"].(string) != "updated-source" {
			t.Errorf("expected updated brief source 'updated-source'")
		}
	})

	t.Run("GET Cache Expired", func(t *testing.T) {
		// Manually update expires_at in DB to 1 hour ago
		err := tx.Model(&models.AIContentBriefCache{}).
			Where("ig_user_id = ?", "123").
			Update("expires_at", time.Now().Add(-1*time.Hour)).Error
		if err != nil {
			t.Fatalf("failed to update expires_at: %v", err)
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/marketing/content-brief-cache?ig_user_id=123", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["cached"].(bool) != false {
			t.Errorf("expected cached false for expired cache")
		}
		if resp["data"] != nil {
			t.Errorf("expected data nil for expired cache")
		}
	})

	t.Run("DELETE Cache - Unauthorized/Forbidden", func(t *testing.T) {
		w := performJSONRequest(t, r, "DELETE", "/api/marketing/content-brief-cache?ig_user_id=123", nil, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}

		w2 := performJSONRequest(t, r, "DELETE", "/api/marketing/content-brief-cache?ig_user_id=123", nil, regularToken)
		if w2.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w2.Code)
		}
	})

	t.Run("DELETE Cache - Success", func(t *testing.T) {
		w := performJSONRequest(t, r, "DELETE", "/api/marketing/content-brief-cache?ig_user_id=123", nil, marketingToken)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["message"].(string) != "Content brief cache deleted" {
			t.Errorf("expected message 'Content brief cache deleted'")
		}

		// Verify GET returns miss
		wGet := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/marketing/content-brief-cache?ig_user_id=123", nil)
		r.ServeHTTP(wGet, req)

		var respGet map[string]interface{}
		json.Unmarshal(wGet.Body.Bytes(), &respGet)
		if respGet["cached"].(bool) != false {
			t.Errorf("expected cached false after delete")
		}
	})
}
