package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"begmt2/config"
	"begmt2/models"
	"begmt2/services"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PreorderController struct {
	cfg      config.Config
	db       *gorm.DB
	hub      *services.NotificationHub
	midtrans services.MidtransService
}

func NewPreorderController(cfg config.Config, db *gorm.DB, hub *services.NotificationHub) PreorderController {
	return PreorderController{
		cfg:      cfg,
		db:       db,
		hub:      hub,
		midtrans: services.NewMidtransService(cfg),
	}
}

type preorderItemReq struct {
	IDProduct       uint    `json:"id_product" binding:"required"`
	Qty             int     `json:"qty" binding:"required,min=1"`
	DiscountPercent float64 `json:"discount_percent" binding:"min=0,max=100"`
}

type createPreorderReq struct {
	NamaCustomer string            `json:"nama_customer" binding:"required,max=255"`
	Email        string            `json:"email" binding:"required,email"`
	Alamat       string            `json:"alamat" binding:"required"`
	NoHP         string            `json:"no_hp" binding:"required,max=50"`
	Catatan      string            `json:"catatan"`
	Items        []preorderItemReq `json:"items" binding:"required,min=1"`
}

type updatePreorderStatusRequest struct {
	Status        models.PreorderStatus `json:"status" binding:"required"`
	InvalidReason *string               `json:"invalid_reason"`
}

func (p PreorderController) CreatePreorder(c *gin.Context) {
	var req createPreorderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	agentID := c.GetUint("user_id")

	preorder := models.Preorder{
		IDAgent:      agentID,
		NamaCustomer: req.NamaCustomer,
		Email:        req.Email,
		Alamat:       req.Alamat,
		NoHP:         req.NoHP,
		Catatan:      req.Catatan,
		Status:       models.PreorderStatusDraft,
	}

	var items []models.PreorderItem
	var subtotal, totalDiscount, total, totalKomisi int64

	err := p.db.Transaction(func(tx *gorm.DB) error {
		for _, itemReq := range req.Items {
			var product models.Product
			if err := tx.First(&product, "id_product = ?", itemReq.IDProduct).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("product not found: ID %d", itemReq.IDProduct)
				}
				return err
			}

			itemSubtotal := product.Price * int64(itemReq.Qty)
			itemDiscountAmount := int64(math.Round(float64(itemSubtotal) * itemReq.DiscountPercent / 100))
			itemTotal := itemSubtotal - itemDiscountAmount
			itemKomisi := int64(math.Round(float64(itemTotal) * p.cfg.AgentCommissionPercent / 100))

			subtotal += itemSubtotal
			totalDiscount += itemDiscountAmount
			total += itemTotal
			totalKomisi += itemKomisi

			items = append(items, models.PreorderItem{
				IDProduct:                  itemReq.IDProduct,
				ProductNameSnapshot:        product.NameProduct,
				ProductPhotoSnapshot:       product.Photo,
				ProductDescriptionSnapshot: product.Description,
				UnitSnapshot:               product.Unit,
				UnitPrice:                  product.Price,
				Qty:                        itemReq.Qty,
				DiscountPercent:            itemReq.DiscountPercent,
				DiscountAmount:             itemDiscountAmount,
				Subtotal:                   itemSubtotal,
				Total:                      itemTotal,
				Komisi:                     itemKomisi,
			})
		}

		preorder.Subtotal = subtotal
		preorder.TotalDiscount = totalDiscount
		preorder.Total = total
		preorder.TotalKomisi = totalKomisi

		if err := tx.Omit("Items").Create(&preorder).Error; err != nil {
			return err
		}

		// PO-1008 for ID 12 => 996 + preorder.ID
		preorder.PONumber = fmt.Sprintf("PO-%04d", 996+preorder.ID)
		if err := tx.Model(&preorder).Update("po_number", preorder.PONumber).Error; err != nil {
			return err
		}

		for i := range items {
			items[i].IDPreorder = preorder.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}

		preorder.Items = items
		return nil
	})

	if err != nil {
		if strings.HasPrefix(err.Error(), "product not found") {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create preorder", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Preorder created",
		"preorder": gin.H{
			"id":             preorder.ID,
			"po_number":      preorder.PONumber,
			"status":         preorder.Status,
			"payment_status": preorder.PaymentStatus,
			"subtotal":       preorder.Subtotal,
			"total_discount": preorder.TotalDiscount,
			"total":          preorder.Total,
			"total_komisi":   preorder.TotalKomisi,
		},
	})
}

func (p PreorderController) ListPreorders(c *gin.Context) {
	search := c.Query("search")
	status := c.Query("status")

	query := p.db.Preload("Agent").Preload("Items").Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Joins("LEFT JOIN preorder_items ON preorder_items.id_preorder = preorders.id").
			Where("preorders.nama_customer LIKE ? OR preorders.email LIKE ? OR preorders.no_hp LIKE ? OR preorder_items.product_name_snapshot LIKE ?", like, like, like, like).
			Distinct()
	}

	var preorders []models.Preorder
	if err := query.Find(&preorders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get preorders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"preorders": preorders})
}

func (p PreorderController) ListAgentPreorders(c *gin.Context) {
	agentID := c.GetUint("user_id")
	status := c.Query("status")

	query := p.db.Preload("Items").Where("id_agent = ?", agentID).Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var preorders []models.Preorder
	if err := query.Find(&preorders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get preorders", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"preorders": preorders})
}

func (p PreorderController) GetPreorder(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	var preorder models.Preorder
	if err := p.db.Preload("Agent").Preload("Items").First(&preorder, preorderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "preorder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get preorder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"preorder": preorder})
}

func (p PreorderController) UpdatePreorder(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	var req createPreorderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	agentID := c.GetUint("user_id")
	var preorder models.Preorder

	err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Items").First(&preorder, preorderID).Error; err != nil {
			return err
		}

		if preorder.IDAgent != agentID {
			return errors.New("unauthorized: you do not own this preorder")
		}

		if preorder.Status != models.PreorderStatusDraft {
			return errors.New("only draft preorder can be updated")
		}

		// Delete existing preorder items
		if err := tx.Where("id_preorder = ?", preorder.ID).Delete(&models.PreorderItem{}).Error; err != nil {
			return err
		}

		var items []models.PreorderItem
		var subtotal, totalDiscount, total, totalKomisi int64

		for _, itemReq := range req.Items {
			var product models.Product
			if err := tx.First(&product, "id_product = ?", itemReq.IDProduct).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("product not found: ID %d", itemReq.IDProduct)
				}
				return err
			}

			itemSubtotal := product.Price * int64(itemReq.Qty)
			itemDiscountAmount := int64(math.Round(float64(itemSubtotal) * itemReq.DiscountPercent / 100))
			itemTotal := itemSubtotal - itemDiscountAmount
			itemKomisi := int64(math.Round(float64(itemTotal) * p.cfg.AgentCommissionPercent / 100))

			subtotal += itemSubtotal
			totalDiscount += itemDiscountAmount
			total += itemTotal
			totalKomisi += itemKomisi

			items = append(items, models.PreorderItem{
				IDPreorder:                 preorder.ID,
				IDProduct:                  itemReq.IDProduct,
				ProductNameSnapshot:        product.NameProduct,
				ProductPhotoSnapshot:       product.Photo,
				ProductDescriptionSnapshot: product.Description,
				UnitSnapshot:               product.Unit,
				UnitPrice:                  product.Price,
				Qty:                        itemReq.Qty,
				DiscountPercent:            itemReq.DiscountPercent,
				DiscountAmount:             itemDiscountAmount,
				Subtotal:                   itemSubtotal,
				Total:                      itemTotal,
				Komisi:                     itemKomisi,
			})
		}

		preorder.NamaCustomer = req.NamaCustomer
		preorder.Email = req.Email
		preorder.Alamat = req.Alamat
		preorder.NoHP = req.NoHP
		preorder.Catatan = req.Catatan
		preorder.Subtotal = subtotal
		preorder.TotalDiscount = totalDiscount
		preorder.Total = total
		preorder.TotalKomisi = totalKomisi
		preorder.PaymentStatus = models.PaymentStatusUnpaid
		preorder.PaymentURL = ""
		preorder.PaymentToken = ""
		preorder.MidtransOrderID = ""
		preorder.MidtransTransactionID = ""

		if err := tx.Omit("Items").Save(&preorder).Error; err != nil {
			return err
		}

		for i := range items {
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}

		preorder.Items = items
		return nil
	})

	if err != nil {
		if err.Error() == "unauthorized: you do not own this preorder" {
			c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
			return
		}
		if err.Error() == "only draft preorder can be updated" {
			c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
			return
		}
		if strings.HasPrefix(err.Error(), "product not found") {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update preorder", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "preorder updated", "preorder": preorder})
}

func (p PreorderController) DeletePreorder(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	agentID := c.GetUint("user_id")
	var preorder models.Preorder

	err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&preorder, preorderID).Error; err != nil {
			return err
		}

		if preorder.IDAgent != agentID {
			return errors.New("unauthorized: you do not own this preorder")
		}

		if preorder.Status != models.PreorderStatusDraft {
			return errors.New("only draft preorder can be deleted")
		}

		if err := tx.Delete(&preorder).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "preorder not found"})
			return
		}
		if err.Error() == "unauthorized: you do not own this preorder" {
			c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
			return
		}
		if err.Error() == "only draft preorder can be deleted" {
			c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete preorder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "preorder deleted"})
}

func (p PreorderController) SubmitPreorder(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	agentID := c.GetUint("user_id")
	var preorder models.Preorder

	err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&preorder, preorderID).Error; err != nil {
			return err
		}

		if preorder.IDAgent != agentID {
			return errors.New("unauthorized: you do not own this preorder")
		}

		if preorder.Status != models.PreorderStatusDraft {
			return errPreorderNotDraft
		}

		if err := tx.Model(&preorder).Update("status", models.PreorderStatusInReview).Error; err != nil {
			return err
		}
		preorder.Status = models.PreorderStatusInReview

		return p.notifySales(tx, preorder)
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "preorder not found"})
			return
		}
		if err.Error() == "unauthorized: you do not own this preorder" {
			c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
			return
		}
		if errors.Is(err, errPreorderNotDraft) {
			c.JSON(http.StatusConflict, gin.H{"message": "only draft preorder can be submitted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to submit preorder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preorder submitted",
		"preorder": gin.H{
			"id":        preorder.ID,
			"po_number": preorder.PONumber,
			"status":    preorder.Status,
		},
	})
}

func (p PreorderController) UpdatePreorderStatus(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	var req updatePreorderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	if req.Status != models.PreorderStatusApprove && req.Status != models.PreorderStatusInvalid {
		c.JSON(http.StatusBadRequest, gin.H{"message": "status must be approve or invalid"})
		return
	}

	var preorder models.Preorder
	err := p.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Items").First(&preorder, preorderID).Error; err != nil {
			return err
		}
		if preorder.Status != models.PreorderStatusInReview {
			return errPreorderNotInReview
		}

		updates := map[string]interface{}{
			"status":         req.Status,
			"invalid_reason": req.InvalidReason,
		}
		if err := tx.Model(&preorder).Updates(updates).Error; err != nil {
			return err
		}

		preorder.Status = req.Status
		if req.InvalidReason != nil {
			preorder.InvalidReason = req.InvalidReason
		}

		if req.Status == models.PreorderStatusApprove {
			if err := p.addPreorderCommissionToWallet(tx, preorder); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "preorder not found"})
			return
		}
		if errors.Is(err, errPreorderNotInReview) {
			c.JSON(http.StatusConflict, gin.H{"message": "only in_review preorder can be approved or invalidated"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update preorder status", "error": err.Error()})
		return
	}

	// For salesSSE notification or websocket compatibility we can just return
	c.JSON(http.StatusOK, gin.H{"message": "preorder status updated", "preorder": preorder})
}

func (p PreorderController) StreamSalesNotifications(c *gin.Context) {
	ch := p.hub.Subscribe(string(models.RoleSales))
	defer p.hub.Unsubscribe(string(models.RoleSales), ch)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-ch:
			c.SSEvent("notification", event)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func (p PreorderController) GetPreorderPDF(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	agentID := c.GetUint("user_id")

	var preorder models.Preorder
	if err := p.db.Preload("Agent").Preload("Items").First(&preorder, preorderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "preorder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get preorder"})
		return
	}

	if preorder.IDAgent != agentID {
		c.JSON(http.StatusForbidden, gin.H{"message": "unauthorized: you do not own this preorder"})
		return
	}

	if preorder.PaymentURL == "" && p.midtrans.IsEnabled() {
		if err := p.ensurePaymentLink(&preorder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create payment link", "error": err.Error()})
			return
		}
	}

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()
	drawQuotationLetterhead(pdf)

	// Title
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "QUOTATION")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, fmt.Sprintf("PO Number: %s", preorder.PONumber))
	pdf.Ln(5)
	if preorder.Agent != nil {
		pdf.Cell(0, 5, fmt.Sprintf("Agent Name: %s", preorder.Agent.Name))
		pdf.Ln(5)
	}
	pdf.Cell(0, 5, fmt.Sprintf("Date: %s", preorder.CreatedAt.Format("2006-01-02 15:04:05")))
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Status: %s", preorder.Status))
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Payment Status: %s", preorder.PaymentStatus))
	pdf.Ln(5)
	if preorder.PaymentURL != "" {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(0, 5, "Official Payment Link:")
		pdf.Ln(5)
		pdf.SetFont("Arial", "U", 9)
		pdf.SetTextColor(28, 75, 151)
		pdf.WriteLinkString(5, preorder.PaymentURL, preorder.PaymentURL)
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(6)
		pdf.SetFont("Arial", "", 8)
		pdf.Cell(0, 5, "Pembayaran hanya melalui link resmi Midtrans di atas.")
		pdf.Ln(5)
	}
	pdf.Ln(10)

	// Customer Details
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Customer Details")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, fmt.Sprintf("Name: %s", preorder.NamaCustomer))
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Email: %s", preorder.Email))
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Phone: %s", preorder.NoHP))
	pdf.Ln(5)
	pdf.Cell(0, 5, fmt.Sprintf("Address: %s", preorder.Alamat))
	pdf.Ln(10)

	if preorder.Catatan != "" {
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 8, "Notes")
		pdf.Ln(8)
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(0, 5, preorder.Catatan)
		pdf.Ln(10)
	}

	// Items Table Header
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(60, 8, "Product Name", "1", 0, "L", false, 0, "")
	pdf.CellFormat(15, 8, "Qty", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 8, "Price", "1", 0, "R", false, 0, "")
	pdf.CellFormat(25, 8, "Discount %", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 8, "Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, "Commission", "1", 0, "R", false, 0, "")
	pdf.Ln(8)

	// Items Table Content
	pdf.SetFont("Arial", "", 9)
	for _, item := range preorder.Items {
		pdf.CellFormat(60, 8, item.ProductNameSnapshot, "1", 0, "L", false, 0, "")
		pdf.CellFormat(15, 8, fmt.Sprintf("%d", item.Qty), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%d", item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(25, 8, fmt.Sprintf("%.1f%%", item.DiscountPercent), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%d", item.Total), "1", 0, "R", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("%d", item.Komisi), "1", 0, "R", false, 0, "")
		pdf.Ln(8)
	}

	// Totals
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(130, 8, "Subtotal", "0", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("%d", preorder.Subtotal), "1", 0, "R", false, 0, "")
	pdf.Ln(8)
	pdf.CellFormat(130, 8, "Total Discount", "0", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("%d", preorder.TotalDiscount), "1", 0, "R", false, 0, "")
	pdf.Ln(8)
	pdf.CellFormat(130, 8, "Total PO", "0", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("%d", preorder.Total), "1", 0, "R", false, 0, "")
	pdf.Ln(8)
	pdf.CellFormat(130, 8, "Total Commission", "0", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("%d", preorder.TotalKomisi), "1", 0, "R", false, 0, "")
	pdf.Ln(8)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s.pdf", preorder.PONumber))
	err := pdf.Output(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate PDF", "error": err.Error()})
	}
}

func (p PreorderController) CreatePreorderPaymentLink(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	agentID := c.GetUint("user_id")
	var preorder models.Preorder
	if err := p.db.Preload("Items").First(&preorder, preorderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "preorder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get preorder"})
		return
	}

	if preorder.IDAgent != agentID {
		c.JSON(http.StatusForbidden, gin.H{"message": "unauthorized: you do not own this preorder"})
		return
	}

	if err := p.ensurePaymentLink(&preorder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create payment link", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "payment link ready",
		"payment": gin.H{
			"payment_status":      preorder.PaymentStatus,
			"payment_url":         preorder.PaymentURL,
			"payment_token":       preorder.PaymentToken,
			"midtrans_order_id":   preorder.MidtransOrderID,
			"midtrans_client_key": p.cfg.MidtransClientKey,
			"environment":         p.cfg.MidtransEnvironment,
		},
	})
}

func (p PreorderController) MidtransNotification(c *gin.Context) {
	var notification services.MidtransNotification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid notification", "error": err.Error()})
		return
	}

	if !p.midtrans.VerifyNotificationSignature(notification) {
		c.JSON(http.StatusForbidden, gin.H{"message": "invalid midtrans signature"})
		return
	}

	paymentStatus := p.midtrans.MapPaymentStatus(notification)
	result := p.db.Model(&models.Preorder{}).
		Where("midtrans_order_id = ?", notification.OrderID).
		Updates(map[string]interface{}{
			"payment_status":          paymentStatus,
			"midtrans_transaction_id": notification.TransactionID,
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update payment status", "error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "preorder payment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification processed"})
}

func (p PreorderController) ensurePaymentLink(preorder *models.Preorder) error {
	if preorder.PaymentURL != "" {
		return nil
	}
	if !p.midtrans.IsEnabled() {
		return errors.New("midtrans is not configured")
	}

	orderID := preorder.MidtransOrderID
	if orderID == "" {
		orderID = fmt.Sprintf("BEGMT2-%s-%d-%d", preorder.PONumber, preorder.ID, time.Now().Unix())
	}

	snapResp, err := p.midtrans.CreateSnapTransaction(
		orderID,
		preorder.Total,
		services.MidtransCustomer{
			FirstName: preorder.NamaCustomer,
			Email:     preorder.Email,
			Phone:     preorder.NoHP,
		},
		[]services.MidtransItem{
			{
				ID:       preorder.PONumber,
				Price:    preorder.Total,
				Quantity: 1,
				Name:     fmt.Sprintf("Quotation %s", preorder.PONumber),
			},
		},
	)
	if err != nil {
		return err
	}

	preorder.MidtransOrderID = orderID
	preorder.PaymentToken = snapResp.Token
	preorder.PaymentURL = snapResp.RedirectURL
	preorder.PaymentStatus = models.PaymentStatusPending

	return p.db.Model(preorder).Updates(map[string]interface{}{
		"midtrans_order_id": orderID,
		"payment_token":     snapResp.Token,
		"payment_url":       snapResp.RedirectURL,
		"payment_status":    models.PaymentStatusPending,
	}).Error
}

func drawQuotationLetterhead(pdf *gofpdf.Fpdf) {
	const (
		pageW   = 210.0
		headerH = 48.0
	)

	pdf.SetAutoPageBreak(true, 12)
	pdf.SetFillColor(244, 248, 235)
	pdf.Rect(0, 0, pageW, headerH, "F")

	// Layered right-side bands approximate the supplied GMT letterhead image.
	bands := []struct {
		x       float64
		r, g, b int
	}{
		{70, 190, 216, 91},
		{92, 159, 197, 77},
		{114, 118, 170, 95},
		{136, 78, 142, 123},
		{158, 52, 102, 153},
		{180, 42, 70, 143},
	}
	for _, band := range bands {
		pdf.SetFillColor(band.r, band.g, band.b)
		pdf.Polygon([]gofpdf.PointType{
			{X: band.x, Y: 0},
			{X: band.x + 36, Y: 0},
			{X: band.x + 7, Y: headerH},
			{X: band.x - 29, Y: headerH},
		}, "F")
	}

	pdf.SetFillColor(255, 255, 255)
	pdf.Polygon([]gofpdf.PointType{
		{X: 0, Y: 45},
		{X: 64, Y: 45},
		{X: 84, Y: 42},
		{X: 101, Y: 34},
		{X: 118, Y: 25},
		{X: 140, Y: 20},
		{X: 210, Y: 20},
		{X: 210, Y: 30},
		{X: 140, Y: 30},
		{X: 119, Y: 34},
		{X: 101, Y: 43},
		{X: 85, Y: 51},
		{X: 0, Y: 51},
	}, "F")

	// Logo mark.
	pdf.SetDrawColor(31, 94, 139)
	pdf.SetLineWidth(0.7)
	for i := 0; i < 12; i++ {
		x := 17.0 + float64(i)*2.1
		pdf.Line(x, 29, x+18, 7)
	}
	pdf.SetDrawColor(114, 184, 67)
	for i := 0; i < 10; i++ {
		x := 12.0 + float64(i)*2.2
		pdf.Line(x, 31, x+18, 9)
	}
	pdf.SetDrawColor(255, 255, 255)
	pdf.SetLineWidth(2.2)
	pdf.Circle(26, 24, 13.5, "D")

	pdf.SetTextColor(28, 75, 151)
	pdf.SetFont("Arial", "", 29)
	pdf.Text(45, 27, "gmt")
	pdf.SetFont("Arial", "", 5.2)
	pdf.Text(45.5, 36, "GLOBAL MULTIPRO TECHNOLOGY")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetDrawColor(255, 255, 255)
	pdf.SetLineWidth(0.35)
	pdf.Circle(116, 32, 2.4, "D")
	pdf.SetFont("Arial", "B", 7.5)
	pdf.Text(121, 33.7, "Rukan Crown Blok B25, Cipondoh, Tangerang")
	pdf.SetFont("Arial", "", 5)
	pdf.Text(115.2, 33.5, "o")

	pdf.Circle(116, 40, 2.4, "D")
	pdf.SetFont("Arial", "B", 7.5)
	pdf.Text(121, 41.7, "+62 852-1567-6696")
	pdf.SetFont("Arial", "", 5)
	pdf.Text(114.7, 41.7, "c")

	pdf.SetY(headerH)
}

func (p PreorderController) notifySales(tx *gorm.DB, preorder models.Preorder) error {
	data, err := json.Marshal(gin.H{
		"id_preorder": preorder.ID,
		"id_agent":    preorder.IDAgent,
		"status":      preorder.Status,
	})
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Preorder #%d masuk untuk review sales", preorder.ID)
	notification := models.Notification{
		Role:    models.RoleSales,
		Title:   "Preorder Baru",
		Message: message,
		Data:    string(data),
	}
	if err := tx.Create(&notification).Error; err != nil {
		return err
	}

	p.hub.Publish(services.NotificationEvent{
		Role:    string(models.RoleSales),
		Title:   notification.Title,
		Message: notification.Message,
		Data:    notification.Data,
	})

	return nil
}

func (p PreorderController) addPreorderCommissionToWallet(tx *gorm.DB, preorder models.Preorder) error {
	var wallet models.AgentWallet
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", preorder.IDAgent).
		First(&wallet).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		wallet = models.AgentWallet{UserID: preorder.IDAgent}
		if err := tx.Create(&wallet).Error; err != nil {
			return err
		}
	}

	productName := fmt.Sprintf("Preorder %s", preorder.PONumber)

	commission := models.AgentCommission{
		UserID:            preorder.IDAgent,
		ProductName:       productName,
		ProductPrice:      preorder.Subtotal,
		DiscountAmount:    preorder.TotalDiscount,
		FinalPrice:        preorder.Total,
		CommissionPercent: p.cfg.AgentCommissionPercent,
		CommissionAmount:  preorder.TotalKomisi,
	}
	if err := tx.Create(&commission).Error; err != nil {
		return err
	}

	return tx.Model(&wallet).Updates(map[string]interface{}{
		"total_commission":  wallet.TotalCommission + preorder.TotalKomisi,
		"available_balance": wallet.AvailableBalance + preorder.TotalKomisi,
	}).Error
}

func (p PreorderController) handlePreorderError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "product not found"})
		return
	}
	if errors.Is(err, errAgentRequired) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id_agent is required"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to process preorder"})
}

func parseUintParam(c *gin.Context, name, message string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": message})
		return 0, false
	}

	return uint(id), true
}

var (
	errAgentRequired       = errors.New("agent required")
	errPreorderNotDraft    = errors.New("preorder is not draft")
	errPreorderNotInReview = errors.New("preorder is not in review")
)
