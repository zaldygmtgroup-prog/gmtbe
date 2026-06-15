package controllers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"begmt2/config"
	"begmt2/controllers"
	"begmt2/models"
	"begmt2/services"

	"github.com/gin-gonic/gin"
)

func TestPreorderMultiProductAndWithdraw(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	tx := db.Begin()
	defer tx.Rollback()

	// Seed products
	p1 := models.Product{
		NameProduct: "Product One",
		Price:       100000,
		Unit:        "pcs",
	}
	p2 := models.Product{
		NameProduct: "Product Two",
		Price:       200000,
		Unit:        "pcs",
	}
	tx.Create(&p1)
	tx.Create(&p2)

	// Seed agent
	agentUser := models.User{
		Name:  "Agent PO Owner",
		Email: "poagent@example.com",
		Role:  models.RoleAgent,
	}
	tx.Create(&agentUser)

	// Seed agent wallet
	wallet := models.AgentWallet{
		UserID:           agentUser.ID,
		AvailableBalance: 1000000,
	}
	tx.Create(&wallet)

	hub := services.NewNotificationHub()
	preorderCtrl := controllers.NewPreorderController(controllers_test_config(), tx, hub)
	agentCtrl := controllers.NewAgentController(controllers_test_config(), tx)

	// 1. Test CreatePreorder Multi-Product
	var preorderID uint
	var poNumber string
	t.Run("CreatePreorder - Success", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", agentUser.ID)
			c.Next()
		})
		r.POST("/api/preorders", preorderCtrl.CreatePreorder)

		body := map[string]interface{}{
			"nama_customer": "Customer Multi",
			"email":         "customer@multi.com",
			"alamat":        "Jl. Multi No. 1",
			"no_hp":         "0811223344",
			"catatan":       "Test multi-product PO",
			"items": []map[string]interface{}{
				{
					"id_product":       p1.IDProduct,
					"qty":              2,
					"discount_percent": 10.0,
				},
				{
					"id_product":       p2.IDProduct,
					"qty":              1,
					"discount_percent": 5.0,
				},
			},
		}
		b, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/preorders", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		po := resp["preorder"].(map[string]interface{})
		preorderID = uint(po["id"].(float64))
		poNumber = po["po_number"].(string)

		if po["subtotal"].(float64) != 400000 {
			t.Errorf("expected subtotal 400,000, got %v", po["subtotal"])
		}
		if po["total_discount"].(float64) != 30000 {
			t.Errorf("expected total_discount 30,000, got %v", po["total_discount"])
		}
		if po["total"].(float64) != 370000 {
			t.Errorf("expected total 370,000, got %v", po["total"])
		}
		if po["total_komisi"].(float64) != 15000 { // Dynamic commission formula
			t.Errorf("expected total_komisi 15,000, got %v", po["total_komisi"])
		}
		if !strings.HasPrefix(poNumber, "PO-") {
			t.Errorf("expected po_number to start with PO-, got %s", poNumber)
		}
	})

	// 2. Test GetPreorder details with items list
	t.Run("GetPreorder - Includes Items", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/preorders/:id", preorderCtrl.GetPreorder)

		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/preorders/%d", preorderID), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		po := resp["preorder"].(map[string]interface{})
		items := po["items"].([]interface{})

		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}
	})

	// 3. Test GetPreorderPDF (Content-Type header)
	t.Run("GetPreorderPDF - Serves PDF", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", agentUser.ID)
			c.Next()
		})
		r.GET("/api/preorders/:id/pdf", preorderCtrl.GetPreorderPDF)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/preorders/%d/pdf", preorderID), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/pdf" {
			t.Errorf("expected Content-Type application/pdf, got %s", contentType)
		}
	})

	// 4. Test CreateWithdraw
	t.Run("CreateWithdraw - Success and Wallet Update", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", agentUser.ID)
			c.Next()
		})
		r.POST("/api/agent/withdraws", agentCtrl.CreateWithdraw)

		body := map[string]interface{}{
			"amount": 200000,
		}
		b, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/agent/withdraws", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		wd := resp["withdraw"].(map[string]interface{})
		wlt := resp["wallet"].(map[string]interface{})

		if wd["withdraw_number"].(string) == "" {
			t.Error("expected withdraw_number to be generated")
		}
		if wlt["available_balance"].(float64) != 800000 {
			t.Errorf("expected available balance to decrease to 800,000, got %v", wlt["available_balance"])
		}
		if wlt["pending_withdraw"].(float64) != 200000 {
			t.Errorf("expected pending withdraw to increase to 200,000, got %v", wlt["pending_withdraw"])
		}
	})
}

func controllers_test_config() config.Config {
	return config.Config{
		AgentCommissionPercent: 5.0,
	}
}
