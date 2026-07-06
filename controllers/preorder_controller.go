package controllers

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	cfg config.Config
	db  *gorm.DB
	hub *services.NotificationHub
}

func NewPreorderController(cfg config.Config, db *gorm.DB, hub *services.NotificationHub) PreorderController {
	return PreorderController{
		cfg: cfg,
		db:  db,
		hub: hub,
	}
}

type preorderItemReq struct {
	IDProduct       uint    `json:"id_product" binding:"required"`
	Qty             int     `json:"qty" binding:"required,min=1"`
	DiscountPercent float64 `json:"discount_percent" binding:"min=0,max=100"`
}

type createPreorderReq struct {
	NamaCustomer string             `json:"nama_customer" binding:"required,max=255"`
	Email        string             `json:"email" binding:"required,email"`
	Alamat       string             `json:"alamat" binding:"required"`
	NoHP         string             `json:"no_hp" binding:"required,max=50"`
	Catatan      string             `json:"catatan"`
	PaymentMode  models.PaymentMode `json:"payment_mode"`
	Items        []preorderItemReq  `json:"items" binding:"required,min=1"`
}

type updatePreorderStatusRequest struct {
	Status        models.PreorderStatus `json:"status" binding:"required"`
	InvalidReason *string               `json:"invalid_reason"`
}

type sendPaymentQuotationRequest struct {
	Stage models.PaymentStage `json:"stage" binding:"required"`
}

func (p PreorderController) CreatePreorder(c *gin.Context) {
	var req createPreorderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	paymentMode := normalizePaymentMode(req.PaymentMode)
	if !isValidPaymentMode(paymentMode) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "payment_mode must be full, split, 50%, or 100%"})
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
		PaymentMode:  paymentMode,
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
			itemKomisi := product.CalculateCommission(itemReq.DiscountPercent) * int64(itemReq.Qty)

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

		poNumber, err := nextPONumber(tx, time.Now())
		if err != nil {
			return err
		}
		preorder.PONumber = poNumber

		if err := tx.Omit("Items").Create(&preorder).Error; err != nil {
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
			"payment_mode":   preorder.PaymentMode,
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
	paymentMode := normalizePaymentMode(req.PaymentMode)
	if !isValidPaymentMode(paymentMode) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "payment_mode must be full, split, 50%, or 100%"})
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
			itemKomisi := product.CalculateCommission(itemReq.DiscountPercent) * int64(itemReq.Qty)

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
		preorder.PaymentMode = paymentMode
		preorder.PaymentStatus = models.PaymentStatusUnpaid
		preorder.PaymentProof = ""
		preorder.DPProof = ""
		preorder.RemainingProof = ""
		preorder.LastPaymentStage = ""

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
		if errors.Is(err, errPaymentProofRequired) {
			c.JSON(http.StatusConflict, gin.H{"message": "payment proof must be uploaded before submitting preorder"})
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
	sendApprovalMessage := false
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
			sendApprovalMessage = true
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

	var sendErrMsg string
	if sendApprovalMessage {
		if err := p.sendApprovedPreorderWhatsApp(preorder.ID); err != nil {
			sendErrMsg = err.Error()
		}
	}

	payload, _ := json.Marshal(gin.H{
		"id":             preorder.ID,
		"po_number":      preorder.PONumber,
		"status":         preorder.Status,
		"payment_status": preorder.PaymentStatus,
		"total":          preorder.Total,
		"total_komisi":   preorder.TotalKomisi,
	})
	p.hub.Publish(services.NotificationEvent{
		Role: fmt.Sprintf("agent_%d_preorders", preorder.IDAgent),
		Data: string(payload),
	})

	resp := gin.H{"message": "preorder status updated", "preorder": preorder}
	if sendErrMsg != "" {
		resp["message"] = "preorder status updated, but failed to send whatsapp: " + sendErrMsg
		resp["error"] = sendErrMsg
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// For salesSSE notification or websocket compatibility we can just return
	c.JSON(http.StatusOK, resp)
}

func (p PreorderController) sendApprovedPreorderWhatsApp(preorderID uint) error {
	var preorder models.Preorder
	if err := p.db.Preload("Agent").Preload("Items").First(&preorder, preorderID).Error; err != nil {
		return fmt.Errorf("failed to load preorder: %w", err)
	}

	pancakeSvc := services.NewPancakeService(p.cfg)
	stage := firstPaymentStage(preorder.PaymentMode)
	if err := p.sendPaymentQuotation(preorder, stage, pancakeSvc); err != nil {
		return err
	}
	if err := p.db.Model(&preorder).Update("last_payment_stage", string(stage)).Error; err != nil {
		return fmt.Errorf("failed to update payment quotation stage: %w", err)
	}

	pdfBytes, filename, err := buildPreorderPDFBytes(preorder)
	if err != nil {
		return fmt.Errorf("failed to generate preorder invoice PDF: %w", err)
	}

	if err := pancakeSvc.SendPreorderInvoice(preorder, pdfBytes, filename); err != nil {
		return fmt.Errorf("failed to send invoice PDF via Pancake: %w", err)
	}
	return nil
}

func (p PreorderController) sendPaymentQuotation(preorder models.Preorder, stage models.PaymentStage, pancakeSvc *services.PancakeService) error {
	amount, err := paymentAmountForStage(preorder, stage)
	if err != nil {
		return err
	}
	if err := pancakeSvc.SendPaymentInstructions(preorder, stage, amount); err != nil {
		return fmt.Errorf("failed to send payment instructions via Pancake: %w", err)
	}

	return nil
}

func (p PreorderController) SendPaymentQuotation(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	var req sendPaymentQuotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	req.Stage = models.PaymentStage(strings.ToLower(strings.TrimSpace(string(req.Stage))))
	if !isValidPaymentStage(req.Stage) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "stage must be full, dp, or remaining"})
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
	if preorder.Status != models.PreorderStatusApprove {
		c.JSON(http.StatusConflict, gin.H{"message": "only approved preorder can receive payment quotation"})
		return
	}
	if err := validatePaymentStageForPreorder(preorder, req.Stage); err != nil {
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		return
	}

	pancakeSvc := services.NewPancakeService(p.cfg)
	if err := p.sendPaymentQuotation(preorder, req.Stage, pancakeSvc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to send payment quotation", "error": err.Error()})
		return
	}

	if err := p.db.Model(&preorder).Update("last_payment_stage", string(req.Stage)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "payment quotation sent, but failed to update preorder", "error": err.Error()})
		return
	}
	preorder.LastPaymentStage = string(req.Stage)

	amount, _ := paymentAmountForStage(preorder, req.Stage)
	c.JSON(http.StatusOK, gin.H{
		"message": "payment quotation sent",
		"payment": gin.H{
			"payment_mode": preorder.PaymentMode,
			"stage":        req.Stage,
			"amount":       amount,
		},
	})
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

func (p PreorderController) StreamAgentPreorders(c *gin.Context) {
	agentID := c.GetUint("user_id")
	topic := fmt.Sprintf("agent_%d_preorders", agentID)

	ch := p.hub.Subscribe(topic)
	defer p.hub.Unsubscribe(topic, ch)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-ch:
			c.SSEvent("preorder_updated", json.RawMessage(event.Data))
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

	pdf := buildPreorderPDF(preorder)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s.pdf", sanitizeIdentifier(preorder.PONumber)))
	err := pdf.Output(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate PDF", "error": err.Error()})
	}
}

func buildPreorderPDFBytes(preorder models.Preorder) ([]byte, string, error) {
	var buffer bytes.Buffer
	pdf := buildPreorderPDF(preorder)
	if err := pdf.Output(&buffer); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("%s.pdf", sanitizeIdentifier(preorder.PONumber))
	return buffer.Bytes(), filename, nil
}

func buildPreorderPDF(preorder models.Preorder) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetFooterFunc(func() {
		drawQuotationFooter(pdf)
	})
	pdf.AddPage()
	drawQuotationLetterhead(pdf)

	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "QUOTATION")
	pdf.Ln(9)

	agentName := "-"
	if preorder.Agent != nil {
		agentName = preorder.Agent.Name
	}
	drawInfoRow(pdf, "PO Information", []string{
		fmt.Sprintf("PO Number: %s", preorder.PONumber),
		fmt.Sprintf("Agent Name: %s", agentName),
		fmt.Sprintf("Date: %s", preorder.CreatedAt.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Status: %s", preorder.Status),
		fmt.Sprintf("Payment Status: %s", preorder.PaymentStatus),
	})
	drawInfoRow(pdf, "Customer Details", []string{
		fmt.Sprintf("Name: %s", preorder.NamaCustomer),
		fmt.Sprintf("Email: %s", preorder.Email),
		fmt.Sprintf("Phone: %s", preorder.NoHP),
		fmt.Sprintf("Address: %s", preorder.Alamat),
	})
	pdf.Ln(5)

	if preorder.Catatan != "" {
		pdf.SetFont("Arial", "B", 9)
		pdf.Cell(0, 8, "Notes")
		pdf.Ln(6)
		pdf.SetFont("Arial", "", 8)
		pdf.MultiCell(0, 4, preorder.Catatan, "", "L", false)
		pdf.Ln(4)
	}

	drawPreorderItemsTable(pdf, preorder.Items)

	pdf.Ln(5)
	drawPreorderTotals(pdf, preorder)
	return pdf
}

func (p PreorderController) UploadPaymentProof(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	agentID := c.GetUint("user_id")
	var preorder models.Preorder
	if err := p.db.First(&preorder, preorderID).Error; err != nil {
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
	if preorder.Status != models.PreorderStatusDraft {
		c.JSON(http.StatusConflict, gin.H{"message": "only draft preorder can upload payment proof"})
		return
	}
	stage := paymentStageFromRequest(c, firstPaymentStage(preorder.PaymentMode))
	if !isValidPaymentStage(stage) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "stage must be full, dp, or remaining"})
		return
	}
	if err := validatePaymentStageForPreorder(preorder, stage); err != nil {
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		return
	}

	proofPath, err := saveRequiredTransferProofUpload(c, p.cfg.UploadDir, "payment_proof", "payment_proofs")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payment_proof", "error": err.Error()})
		return
	}

	updates := paymentProofUpdates(stage, proofPath)
	applyPaymentProofToPreorder(&preorder, stage, proofPath)
	if err := p.db.Model(&preorder).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to upload payment proof", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "payment proof uploaded",
		"payment": gin.H{
			"payment_status":  preorder.PaymentStatus,
			"stage":           stage,
			"payment_proof":   preorder.PaymentProof,
			"dp_proof":        preorder.DPProof,
			"remaining_proof": preorder.RemainingProof,
		},
	})
}

func (p PreorderController) UploadSalesPaymentProof(c *gin.Context) {
	preorderID, ok := parseUintParam(c, "id", "invalid preorder id")
	if !ok {
		return
	}

	var preorder models.Preorder
	if err := p.db.First(&preorder, preorderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "preorder not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get preorder"})
		return
	}
	if preorder.Status != models.PreorderStatusApprove {
		c.JSON(http.StatusConflict, gin.H{"message": "only approved preorder can upload payment proof"})
		return
	}

	stage := paymentStageFromRequest(c, firstPaymentStage(preorder.PaymentMode))
	if !isValidPaymentStage(stage) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "stage must be full, dp, or remaining"})
		return
	}
	if err := validatePaymentStageForPreorder(preorder, stage); err != nil {
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		return
	}

	proofPath, err := saveRequiredTransferProofUpload(c, p.cfg.UploadDir, "payment_proof", "payment_proofs")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payment_proof", "error": err.Error()})
		return
	}

	updates := paymentProofUpdates(stage, proofPath)
	applyPaymentProofToPreorder(&preorder, stage, proofPath)
	if err := p.db.Model(&preorder).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to upload payment proof", "error": err.Error()})
		return
	}

	payload, _ := json.Marshal(gin.H{
		"id":             preorder.ID,
		"po_number":      preorder.PONumber,
		"status":         preorder.Status,
		"payment_mode":   preorder.PaymentMode,
		"payment_status": preorder.PaymentStatus,
		"stage":          stage,
	})
	p.hub.Publish(services.NotificationEvent{
		Role: fmt.Sprintf("agent_%d_preorders", preorder.IDAgent),
		Data: string(payload),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "payment proof uploaded",
		"payment": gin.H{
			"payment_mode":    preorder.PaymentMode,
			"payment_status":  preorder.PaymentStatus,
			"stage":           stage,
			"payment_proof":   preorder.PaymentProof,
			"dp_proof":        preorder.DPProof,
			"remaining_proof": preorder.RemainingProof,
		},
	})
}

func drawInfoRow(pdf *gofpdf.Fpdf, title string, lines []string) {
	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	usableW := pageW - left - right
	x := pdf.GetX()
	y := pdf.GetY()
	rowH := 20.0

	pdf.SetDrawColor(180, 180, 180)
	pdf.Rect(x, y, usableW, rowH, "D")
	pdf.SetFillColor(110, 170, 70)
	pdf.Rect(x, y, usableW, 6, "F")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 7)
	pdf.SetXY(x+2, y+1.2)
	pdf.Cell(0, 3.5, title)

	pdf.SetFont("Arial", "", 7)
	pdf.SetXY(x+2, y+8)
	colW := usableW / 3
	lineH := 4.2
	for i, line := range lines {
		col := i % 3
		row := i / 3
		pdf.SetXY(x+2+float64(col)*colW, y+8+float64(row)*lineH)
		pdf.CellFormat(colW-4, lineH, line, "", 0, "L", false, 0, "")
	}
	pdf.SetXY(x, y+rowH+2)
}

func drawPreorderItemsTable(pdf *gofpdf.Fpdf, items []models.PreorderItem) {
	widths := []float64{7, 24, 42, 23, 14, 20, 17, 21, 22}
	headers := []string{"NO", "Model", "Deskripsi Produk", "Picture", "Quantity", "Unit Price", "Discount", "After Discount", "Total Price"}
	aligns := []string{"C", "L", "L", "C", "C", "R", "R", "R", "R"}

	drawPreorderItemsHeader(pdf, widths, headers)

	pdf.SetFont("Arial", "", 6)
	for i, item := range items {
		description := item.ProductDescriptionSnapshot
		if strings.TrimSpace(description) == "" {
			description = "-"
		}
		qtyText := fmt.Sprintf("%d", item.Qty)
		unitDiscount := int64(math.Round(float64(item.UnitPrice) * item.DiscountPercent / 100))
		afterDiscount := item.UnitPrice - unitDiscount
		cells := []string{
			fmt.Sprintf("%d", i+1),
			item.ProductNameSnapshot,
			description,
			"",
			qtyText,
			formatRupiah(item.UnitPrice),
			formatRupiah(unitDiscount),
			formatRupiah(afterDiscount),
			formatRupiah(item.Total),
		}

		rowH := calculatePDFRowHeight(pdf, widths, cells, 3.4, 18)
		ensurePDFSpace(pdf, rowH+20, func() {
			drawPreorderItemsHeader(pdf, widths, headers)
			pdf.SetFont("Arial", "", 6)
		})

		x := pdf.GetX()
		y := pdf.GetY()
		for j, cell := range cells {
			pdf.Rect(x, y, widths[j], rowH, "D")
			if j == 3 {
				drawProductImage(pdf, item.ProductPhotoSnapshot, x+2, y+2, widths[j]-4, rowH-4)
			} else {
				pdf.SetXY(x+1, y+1.2)
				pdf.MultiCell(widths[j]-2, 3.4, cell, "", aligns[j], false)
			}
			x += widths[j]
			pdf.SetXY(x, y)
		}
		pdf.SetXY(10, y+rowH)
	}
}

func drawPreorderItemsHeader(pdf *gofpdf.Fpdf, widths []float64, headers []string) {
	ensurePDFSpace(pdf, 12, nil)
	pdf.SetFillColor(112, 173, 71)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "B", 6)
	for i, header := range headers {
		pdf.CellFormat(widths[i], 6, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(6)
}

func drawPreorderTotals(pdf *gofpdf.Fpdf, preorder models.Preorder) {
	ensurePDFSpace(pdf, 26, nil)
	pageW, _ := pdf.GetPageSize()
	_, _, right, _ := pdf.GetMargins()
	valueW := 46.0
	labelW := 35.0
	labelX := pageW - right - labelW - valueW

	pdf.SetFont("Arial", "B", 7)
	pdf.SetX(labelX)
	pdf.CellFormat(labelW, 5, "SUBTOTAL", "", 0, "L", false, 0, "")
	pdf.CellFormat(valueW, 5, formatRupiah(preorder.Subtotal), "B", 1, "R", false, 0, "")

	pdf.SetX(labelX)
	pdf.CellFormat(labelW, 5, "Discount", "", 0, "L", false, 0, "")
	pdf.CellFormat(valueW, 5, formatRupiah(preorder.TotalDiscount), "B", 1, "R", false, 0, "")

	pdf.SetX(labelX)
	pdf.CellFormat(labelW, 5, "Total", "", 0, "L", false, 0, "")
	pdf.CellFormat(valueW, 5, formatRupiah(preorder.Total), "B", 1, "R", false, 0, "")
}

func calculatePDFRowHeight(pdf *gofpdf.Fpdf, widths []float64, cells []string, lineH float64, minH float64) float64 {
	maxLines := 1
	for i, cell := range cells {
		if i == 3 {
			continue
		}
		lines := pdf.SplitLines([]byte(cell), widths[i]-2)
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	height := float64(maxLines)*lineH + 2.4
	if height < minH {
		return minH
	}
	return height
}

func ensurePDFSpace(pdf *gofpdf.Fpdf, needed float64, afterAddPage func()) {
	_, pageH := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	if pdf.GetY()+needed > pageH-bottom-14 {
		pdf.AddPage()
		drawQuotationLetterhead(pdf)
		if afterAddPage != nil {
			afterAddPage()
		}
	}
}

func drawProductImage(pdf *gofpdf.Fpdf, imagePath string, x, y, maxW, maxH float64) {
	if drawRemoteProductImage(pdf, imagePath, x, y, maxW, maxH) {
		return
	}
	path, ok := resolvePDFAssetPath(imagePath)
	if !ok {
		return
	}
	drawLocalImageFit(pdf, path, x, y, maxW, maxH)
}

func drawRemoteProductImage(pdf *gofpdf.Fpdf, imageURL string, x, y, maxW, maxH float64) bool {
	parsed, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}

	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(parsed.String())
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return false
	}

	limit := int64(20 << 20) // 20 MB limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(body) == 0 {
		return false
	}
	if int64(len(body)) > limit {
		// Truncated / too large, skip
		return false
	}

	// Validate that the image can actually be decoded to prevent PDF generation crash
	_, _, err = image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return false
	}

	imageType := imageTypeFromContentType(resp.Header.Get("Content-Type"))
	if imageType == "" {
		imageType = imageTypeFromPath(parsed.Path)
	}
	if imageType == "" {
		imageType = "JPG"
	}

	drawW, drawH, drawX, drawY := fitImageBox(body, x, y, maxW, maxH)
	key := fmt.Sprintf("product-%x", sha1.Sum([]byte(parsed.String())))
	options := gofpdf.ImageOptions{ImageType: imageType}
	pdf.RegisterImageOptionsReader(key, options, bytes.NewReader(body))
	pdf.ImageOptions(key, drawX, drawY, drawW, drawH, false, options, 0, "")
	return true
}

func drawLocalImageFit(pdf *gofpdf.Fpdf, path string, x, y, maxW, maxH float64) {
	body, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_, _, err = image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return
	}
	drawW, drawH, drawX, drawY := fitImageBox(body, x, y, maxW, maxH)
	pdf.ImageOptions(path, drawX, drawY, drawW, drawH, false, gofpdf.ImageOptions{}, 0, "")
}

func fitImageBox(body []byte, x, y, maxW, maxH float64) (float64, float64, float64, float64) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return maxW, maxH, x, y
	}

	ratio := float64(cfg.Width) / float64(cfg.Height)
	drawW := maxW
	drawH := drawW / ratio
	if drawH > maxH {
		drawH = maxH
		drawW = drawH * ratio
	}
	drawX := x + (maxW-drawW)/2
	drawY := y + (maxH-drawH)/2
	return drawW, drawH, drawX, drawY
}

func imageTypeFromContentType(contentType string) string {
	contentType = strings.ToLower(contentType)
	switch {
	case strings.Contains(contentType, "png"):
		return "PNG"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return "JPG"
	case strings.Contains(contentType, "gif"):
		return "GIF"
	default:
		return ""
	}
}

func imageTypeFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "PNG"
	case ".jpg", ".jpeg":
		return "JPG"
	case ".gif":
		return "GIF"
	default:
		return ""
	}
}

func formatRupiah(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	raw := strconv.FormatInt(value, 10)
	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	return fmt.Sprintf("%sRp %s", sign, strings.Join(parts, "."))
}

func resolvePDFAssetPath(assetPath string) (string, bool) {
	if strings.TrimSpace(assetPath) == "" {
		return "", false
	}
	candidates := []string{assetPath}
	if strings.HasPrefix(assetPath, "/") || strings.HasPrefix(assetPath, "\\") {
		trimmed := strings.TrimLeft(assetPath, `/\`)
		candidates = append(candidates,
			filepath.Join(".", trimmed),
			filepath.Join("..", trimmed),
		)
	}
	if !filepath.IsAbs(assetPath) {
		candidates = append(candidates,
			filepath.Join(".", assetPath),
			filepath.Join("..", assetPath),
		)
	}
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, true
		}
	}
	return "", false
}

func drawQuotationLetterhead(pdf *gofpdf.Fpdf) {
	const (
		assetW = 297.0
		assetH = 69.4
	)
	pageW, _ := pdf.GetPageSize()
	headerH := assetH * pageW / assetW

	if path, ok := resolvePDFAssetPath("kop_surat.png"); ok {
		pdf.ImageOptions(path, 0, 0, pageW, headerH, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	pdf.SetY(headerH)
}

func drawQuotationFooter(pdf *gofpdf.Fpdf) {
	const (
		assetW = 297.0
		assetH = 14.4
	)
	path, ok := resolvePDFAssetPath("footer_surat.png")
	if !ok {
		return
	}
	pageW, pageH := pdf.GetPageSize()
	footerH := assetH * pageW / assetW
	pdf.ImageOptions(path, 0, pageH-footerH, pageW, footerH, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
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

func nextPONumber(tx *gorm.DB, now time.Time) (string, error) {
	prefix := now.Format("INV/GMT/2006/01/")

	var lastPreorder models.Preorder
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("po_number LIKE ?", "INV/GMT/%").
		Order("id DESC").
		First(&lastPreorder).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	nextSequence := 1
	if err == nil {
		parts := strings.Split(lastPreorder.PONumber, "/")
		if len(parts) > 0 {
			lastSequence, parseErr := strconv.Atoi(parts[len(parts)-1])
			if parseErr == nil {
				nextSequence = lastSequence + 1
			}
		}
	}

	return fmt.Sprintf("%s%04d", prefix, nextSequence), nil
}

func sanitizeIdentifier(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-")
	return replacer.Replace(value)
}

func normalizePaymentMode(mode models.PaymentMode) models.PaymentMode {
	val := strings.ToLower(strings.TrimSpace(string(mode)))
	if val == "" || val == "full" || val == "100%" || val == "100" {
		return models.PaymentModeFull
	}
	if val == "split" || val == "50%" || val == "50" {
		return models.PaymentModeSplit
	}
	return models.PaymentMode(val)
}

func isValidPaymentMode(mode models.PaymentMode) bool {
	return mode == models.PaymentModeFull || mode == models.PaymentModeSplit
}

func isValidPaymentStage(stage models.PaymentStage) bool {
	return stage == models.PaymentStageFull || stage == models.PaymentStageDP || stage == models.PaymentStageRemaining
}

func firstPaymentStage(mode models.PaymentMode) models.PaymentStage {
	if mode == models.PaymentModeSplit {
		return models.PaymentStageDP
	}
	return models.PaymentStageFull
}

func paymentStageFromRequest(c *gin.Context, fallback models.PaymentStage) models.PaymentStage {
	stage := strings.TrimSpace(c.PostForm("stage"))
	if stage == "" {
		stage = strings.TrimSpace(c.Query("stage"))
	}
	if stage == "" {
		return fallback
	}
	return models.PaymentStage(strings.ToLower(stage))
}

func validatePaymentStageForPreorder(preorder models.Preorder, stage models.PaymentStage) error {
	mode := normalizePaymentMode(preorder.PaymentMode)
	if mode == models.PaymentModeFull {
		if stage != models.PaymentStageFull {
			return fmt.Errorf("full payment preorder only accepts full payment stage")
		}
		return nil
	}

	switch stage {
	case models.PaymentStageDP:
		return nil
	case models.PaymentStageRemaining:
		if strings.TrimSpace(preorder.DPProof) == "" {
			return fmt.Errorf("dp payment proof must be uploaded before remaining payment")
		}
		return nil
	default:
		return fmt.Errorf("split payment preorder only accepts dp or remaining payment stage")
	}
}

func paymentAmountForStage(preorder models.Preorder, stage models.PaymentStage) (int64, error) {
	if err := validatePaymentStageForPreorder(preorder, stage); err != nil {
		return 0, err
	}
	if stage == models.PaymentStageDP {
		return preorder.Total / 2, nil
	}
	if stage == models.PaymentStageRemaining {
		return preorder.Total - preorder.Total/2, nil
	}
	return preorder.Total, nil
}

func paymentProofUpdates(stage models.PaymentStage, proofPath string) map[string]interface{} {
	updates := map[string]interface{}{
		"last_payment_stage": string(stage),
	}
	switch stage {
	case models.PaymentStageDP:
		updates["dp_proof"] = proofPath
		updates["payment_status"] = models.PaymentStatusPartial
	case models.PaymentStageRemaining:
		updates["remaining_proof"] = proofPath
		updates["payment_status"] = models.PaymentStatusPaid
	default:
		updates["payment_proof"] = proofPath
		updates["payment_status"] = models.PaymentStatusPaid
	}
	return updates
}

func applyPaymentProofToPreorder(preorder *models.Preorder, stage models.PaymentStage, proofPath string) {
	preorder.LastPaymentStage = string(stage)
	switch stage {
	case models.PaymentStageDP:
		preorder.DPProof = proofPath
		preorder.PaymentStatus = models.PaymentStatusPartial
	case models.PaymentStageRemaining:
		preorder.RemainingProof = proofPath
		preorder.PaymentStatus = models.PaymentStatusPaid
	default:
		preorder.PaymentProof = proofPath
		preorder.PaymentStatus = models.PaymentStatusPaid
	}
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
	errAgentRequired        = errors.New("agent required")
	errPreorderNotDraft     = errors.New("preorder is not draft")
	errPreorderNotInReview  = errors.New("preorder is not in review")
	errPaymentProofRequired = errors.New("payment proof is required")
)
