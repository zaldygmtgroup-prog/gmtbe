package controllers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"begmt2/config"
	"begmt2/controllers"
	"begmt2/middleware"
	"begmt2/models"
	"begmt2/utils"

	"github.com/gin-gonic/gin"
)

func TestAuthSSOFlowAndGlobalLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	tx := db.Begin()
	defer tx.Rollback()

	password, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		Name:        "SSO User",
		TTL:         "-",
		PhoneNumber: "-",
		Gender:      "-",
		Email:       "sso-user@example.com",
		Domicile:    "-",
		Password:    password,
		Role:        models.RoleUser,
		DetailUser: models.DetailUser{
			CompanyName: "-",
		},
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	cfg := config.Config{
		JWTSecret:             "test-secret",
		JWTExpiresHours:       24,
		SSOCodeExpiresSeconds: 60,
		SSOClientRedirects: map[string]string{
			"website_a":     "https://website-a.test/sso/callback",
			"website_utama": "https://website-utama.test/sso/callback",
		},
	}
	ctrl := controllers.NewAuthController(cfg, tx)

	r := gin.New()
	auth := r.Group("/api/auth")
	auth.POST("/login", ctrl.Login)
	auth.POST("/sso/exchange", ctrl.ExchangeSSOCode)
	protected := auth.Group("")
	protected.Use(middleware.AuthMiddleware(cfg, tx))
	protected.GET("/session", ctrl.Session)
	protected.POST("/logout", ctrl.Logout)
	protected.POST("/sso/code", ctrl.CreateSSOCode)

	loginBody := map[string]interface{}{
		"email":    user.Email,
		"password": "password123",
		"client":   "website_a",
	}
	loginResp := performJSONRequest(t, r, http.MethodPost, "/api/auth/login", loginBody, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var loginPayload map[string]interface{}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	sourceToken := loginPayload["token"].(string)

	sessionResp := performJSONRequest(t, r, http.MethodGet, "/api/auth/session", nil, sourceToken)
	if sessionResp.Code != http.StatusOK {
		t.Fatalf("expected session status 200, got %d: %s", sessionResp.Code, sessionResp.Body.String())
	}

	codeBody := map[string]interface{}{
		"target_client": "website_utama",
		"state":         "abc",
	}
	codeResp := performJSONRequest(t, r, http.MethodPost, "/api/auth/sso/code", codeBody, sourceToken)
	if codeResp.Code != http.StatusCreated {
		t.Fatalf("expected sso code status 201, got %d: %s", codeResp.Code, codeResp.Body.String())
	}

	var codePayload map[string]interface{}
	if err := json.Unmarshal(codeResp.Body.Bytes(), &codePayload); err != nil {
		t.Fatalf("failed to decode code response: %v", err)
	}
	code := codePayload["code"].(string)

	exchangeBody := map[string]interface{}{
		"code":          code,
		"target_client": "website_utama",
	}
	exchangeResp := performJSONRequest(t, r, http.MethodPost, "/api/auth/sso/exchange", exchangeBody, "")
	if exchangeResp.Code != http.StatusOK {
		t.Fatalf("expected exchange status 200, got %d: %s", exchangeResp.Code, exchangeResp.Body.String())
	}

	var exchangePayload map[string]interface{}
	if err := json.Unmarshal(exchangeResp.Body.Bytes(), &exchangePayload); err != nil {
		t.Fatalf("failed to decode exchange response: %v", err)
	}
	targetToken := exchangePayload["token"].(string)

	reuseResp := performJSONRequest(t, r, http.MethodPost, "/api/auth/sso/exchange", exchangeBody, "")
	if reuseResp.Code != http.StatusBadRequest {
		t.Fatalf("expected reused code status 400, got %d: %s", reuseResp.Code, reuseResp.Body.String())
	}

	logoutResp := performJSONRequest(t, r, http.MethodPost, "/api/auth/logout", nil, targetToken)
	if logoutResp.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d: %s", logoutResp.Code, logoutResp.Body.String())
	}

	revokedSourceResp := performJSONRequest(t, r, http.MethodGet, "/api/auth/session", nil, sourceToken)
	if revokedSourceResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected original session to be revoked, got %d: %s", revokedSourceResp.Code, revokedSourceResp.Body.String())
	}
}

func performJSONRequest(t *testing.T, r http.Handler, method, path string, body interface{}, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody *bytes.Buffer
	if body == nil {
		requestBody = bytes.NewBuffer(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		requestBody = bytes.NewBuffer(payload)
	}

	req, err := http.NewRequest(method, path, requestBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
