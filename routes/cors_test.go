package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"begmt2/config"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareAllowsOriginConfiguredWithTrailingSlash(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(corsMiddleware(config.Config{
		CORSAllowedOrigins: []string{"https://gmtgroup2.vercel.app/"},
	}))
	r.POST("/api/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	req.Header.Set("Origin", "https://gmtgroup2.vercel.app")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://gmtgroup2.vercel.app" {
		t.Fatalf("expected Access-Control-Allow-Origin for Website A, got %q", got)
	}
}
