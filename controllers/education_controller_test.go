package controllers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"begmt2/config"
	"begmt2/controllers"
	"begmt2/models"

	"github.com/gin-gonic/gin"
)

func TestEducationGetIncludesRegistrationStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	tx := db.Begin()
	defer tx.Rollback()

	suffix := time.Now().Format("20060102150405.000000000")
	educationID := "edu_test_" + suffix
	educationStatus := "TestEducationStatus_" + suffix

	user := models.User{
		Name:        "Education User",
		TTL:         "-",
		PhoneNumber: "-",
		Gender:      "-",
		Email:       "education-user-" + suffix + "@example.com",
		Domicile:    "-",
		Role:        models.RoleUser,
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	education := models.Education{
		ID:          educationID,
		Title:       "GMT Group Education Event",
		Description: "Education description",
		Date:        "2026-08-15",
		Status:      educationStatus,
	}
	if err := tx.Create(&education).Error; err != nil {
		t.Fatalf("failed to create education: %v", err)
	}

	cfg := config.Config{
		JWTSecret:       "test-secret-key-12345",
		JWTExpiresHours: 24,
	}
	token := generateTestToken(t, tx, user, cfg)

	ctrl := controllers.NewEducationController(cfg, tx)
	r := gin.New()
	r.GET("/api/educations", ctrl.ListEducations)
	r.GET("/api/educations/:id", ctrl.GetEducation)

	guestResp := performJSONRequest(t, r, http.MethodGet, "/api/educations/"+educationID, nil, "")
	if guestResp.Code != http.StatusOK {
		t.Fatalf("expected guest status 200, got %d: %s", guestResp.Code, guestResp.Body.String())
	}

	var guestPayload map[string]interface{}
	if err := json.Unmarshal(guestResp.Body.Bytes(), &guestPayload); err != nil {
		t.Fatalf("failed to decode guest response: %v", err)
	}
	guestData := guestPayload["data"].(map[string]interface{})
	if guestData["is_registered"].(bool) {
		t.Fatalf("expected guest is_registered false")
	}

	authRespBeforeRegister := performJSONRequest(t, r, http.MethodGet, "/api/educations/"+educationID, nil, token)
	if authRespBeforeRegister.Code != http.StatusOK {
		t.Fatalf("expected auth status 200, got %d: %s", authRespBeforeRegister.Code, authRespBeforeRegister.Body.String())
	}

	var beforePayload map[string]interface{}
	if err := json.Unmarshal(authRespBeforeRegister.Body.Bytes(), &beforePayload); err != nil {
		t.Fatalf("failed to decode auth response before register: %v", err)
	}
	beforeData := beforePayload["data"].(map[string]interface{})
	if beforeData["is_registered"].(bool) {
		t.Fatalf("expected is_registered false before registration")
	}

	registration := models.EducationRegistration{
		ID:        "reg_test_" + suffix,
		EventID:   education.ID,
		UserID:    user.ID,
		FirstName: "Education",
		Surname:   "User",
		Email:     user.Email,
		Status:    "Confirmed",
	}
	if err := tx.Create(&registration).Error; err != nil {
		t.Fatalf("failed to create registration: %v", err)
	}

	authRespAfterRegister := performJSONRequest(t, r, http.MethodGet, "/api/educations/"+educationID, nil, token)
	if authRespAfterRegister.Code != http.StatusOK {
		t.Fatalf("expected auth status 200, got %d: %s", authRespAfterRegister.Code, authRespAfterRegister.Body.String())
	}

	var afterPayload map[string]interface{}
	if err := json.Unmarshal(authRespAfterRegister.Body.Bytes(), &afterPayload); err != nil {
		t.Fatalf("failed to decode auth response after register: %v", err)
	}
	afterData := afterPayload["data"].(map[string]interface{})
	if !afterData["is_registered"].(bool) {
		t.Fatalf("expected is_registered true after registration")
	}

	listResp := performJSONRequest(t, r, http.MethodGet, "/api/educations?status="+educationStatus, nil, token)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	var listPayload map[string]interface{}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	listData := listPayload["data"].([]interface{})
	if len(listData) != 1 {
		t.Fatalf("expected one education item, got %d", len(listData))
	}
	firstEducation := listData[0].(map[string]interface{})
	if !firstEducation["is_registered"].(bool) {
		t.Fatalf("expected list item is_registered true after registration")
	}
}
