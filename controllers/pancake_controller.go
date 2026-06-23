package controllers

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"begmt2/config"
	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PancakeController struct {
	cfg config.Config
	db  *gorm.DB
	loc *time.Location
}

func NewPancakeController(cfg config.Config, db *gorm.DB) PancakeController {
	loc, err := time.LoadLocation(cfg.AnalyticsTimezone)
	if err != nil {
		loc = time.FixedZone("Asia/Jakarta", 7*60*60)
	}
	return PancakeController{cfg: cfg, db: db, loc: loc}
}

type pancakeWebhookPayload struct {
	PageID    string `json:"page_id"`
	EventType string `json:"event_type" binding:"required"`
	Data      struct {
		Conversation struct {
			ID          string   `json:"id"`
			Type        string   `json:"type"`
			AssigneeIDs []string `json:"assignee_ids"`
			IsReplied   bool     `json:"is_replied"`
			From        struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"from"`
		} `json:"conversation"`
		Message struct {
			ID              string `json:"id"`
			ConversationID  string `json:"conversation_id"`
			PageID          string `json:"page_id"`
			Message         string `json:"message"`
			OriginalMessage string `json:"original_message"`
			Type            string `json:"type"`
			InsertedAt      string `json:"inserted_at"`
			HasPhone        bool   `json:"has_phone"`
			From            struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				PageCustomerID string `json:"page_customer_id"`
			} `json:"from"`
		} `json:"message"`
		Post *struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"post"`
	} `json:"data"`
}

func (p PancakeController) Webhook(c *gin.Context) {
	if p.cfg.PancakeWebhookSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "PANCAKE_WEBHOOK_SECRET is not configured"})
		return
	}
	provided := c.GetHeader("X-Pancake-Webhook-Secret")
	if provided == "" {
		provided = c.Query("secret")
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(p.cfg.PancakeWebhookSecret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid webhook secret"})
		return
	}

	var payload pancakeWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid Pancake webhook payload"})
		return
	}
	if payload.EventType != "messaging" {
		// Subscription and post events are valid but not required for chat analytics.
		c.JSON(http.StatusOK, gin.H{"success": true, "ignored": true})
		return
	}
	if payload.PageID == "" || payload.Data.Message.ID == "" || payload.Data.Conversation.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "page, message and conversation IDs are required"})
		return
	}

	insertedAt, err := parsePancakeTime(payload.Data.Message.InsertedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid message inserted_at"})
		return
	}
	text := payload.Data.Message.Message
	if text == "" {
		text = payload.Data.Message.OriginalMessage
	}
	customerID := payload.Data.Conversation.From.ID
	direction := "page"
	if payload.Data.Message.From.ID == customerID {
		direction = "customer"
	}
	postID, postMessage := "", ""
	if payload.Data.Post != nil {
		postID, postMessage = payload.Data.Post.ID, payload.Data.Post.Message
	}

	err = p.db.Transaction(func(tx *gorm.DB) error {
		var existing models.PancakeMessage
		findErr := tx.First(&existing, "id = ?", payload.Data.Message.ID).Error
		isNew := errors.Is(findErr, gorm.ErrRecordNotFound)
		if findErr != nil && !isNew {
			return findErr
		}

		message := models.PancakeMessage{
			ID: payload.Data.Message.ID, PageID: payload.PageID,
			ConversationID: payload.Data.Conversation.ID, CustomerID: customerID,
			SenderID: payload.Data.Message.From.ID, SenderName: payload.Data.Message.From.Name,
			Direction: direction, Type: payload.Data.Message.Type, Text: text,
			HasPhone: payload.Data.Message.HasPhone, InsertedAt: insertedAt,
		}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&message).Error; err != nil {
			return err
		}

		var conversation models.PancakeConversation
		convFindErr := tx.First(&conversation, "id = ?", payload.Data.Conversation.ID).Error
		convNew := errors.Is(convFindErr, gorm.ErrRecordNotFound)
		if convFindErr != nil && !convNew {
			return convFindErr
		}
		if convNew {
			conversation.ID = payload.Data.Conversation.ID
		}
		conversation.PageID = payload.PageID
		conversation.CustomerID = customerID
		conversation.CustomerName = payload.Data.Conversation.From.Name
		conversation.Type = payload.Data.Conversation.Type
		conversation.PostID = postID
		conversation.PostMessage = postMessage
		if payload.Data.Message.From.PageCustomerID != "" {
			conversation.PageCustomerID = payload.Data.Message.From.PageCustomerID
		}
		if isNew {
			if direction == "customer" {
				conversation.CustomerMessageCount++
				conversation.LastCustomerMessageAt = &insertedAt
				if conversation.FirstCustomerMessageAt == nil || insertedAt.Before(*conversation.FirstCustomerMessageAt) {
					conversation.FirstCustomerMessageAt = &insertedAt
				}
			} else {
				conversation.PageMessageCount++
				conversation.LastPageMessageAt = &insertedAt
			}
		}
		conversation.HasPhone = conversation.HasPhone || payload.Data.Message.HasPhone
		return tx.Save(&conversation).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to store Pancake event"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type conversionRequest struct {
	ExternalOrderID string    `json:"external_order_id" binding:"required"`
	PageID          string    `json:"page_id" binding:"required"`
	ConversationID  string    `json:"conversation_id" binding:"required"`
	CustomerID      string    `json:"customer_id"`
	CampaignID      string    `json:"campaign_id"`
	CampaignName    string    `json:"campaign_name"`
	ProductName     string    `json:"product_name"`
	Amount          int64     `json:"amount"`
	ConvertedAt     time.Time `json:"converted_at"`
}

func (p PancakeController) UpsertConversion(c *gin.Context) {
	var req conversionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "external_order_id, page_id, conversation_id and a valid amount are required"})
		return
	}
	if req.ConvertedAt.IsZero() {
		req.ConvertedAt = time.Now()
	}
	conversion := models.PancakeConversion{
		ExternalOrderID: req.ExternalOrderID, PageID: req.PageID,
		ConversationID: req.ConversationID, CustomerID: req.CustomerID,
		CampaignID: req.CampaignID, CampaignName: req.CampaignName,
		ProductName: req.ProductName, Amount: req.Amount, ConvertedAt: req.ConvertedAt,
	}
	if err := p.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "external_order_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"page_id", "conversation_id", "customer_id", "campaign_id", "campaign_name", "product_name", "amount", "converted_at", "updated_at"}),
	}).Create(&conversion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save conversion"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "conversion": conversion})
}

func parsePancakeTime(value string) (time.Time, error) {
	formats := []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999", "2006-01-02T15:04:05"}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, errors.New("unsupported time format")
}

type rankedItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type customerScore struct {
	ConversationID string     `json:"conversation_id"`
	CustomerID     string     `json:"customer_id"`
	CustomerName   string     `json:"customer_name"`
	Score          int        `json:"score"`
	Reasons        []string   `json:"reasons"`
	LastChatAt     *time.Time `json:"last_chat_at"`
}

func (p PancakeController) Analytics(c *gin.Context) {
	from, to, ok := p.analyticsRange(c)
	if !ok {
		return
	}
	pageID := c.Query("page_id")

	convQuery := p.db.Where("first_customer_message_at IS NOT NULL")
	msgQuery := p.db.Where("direction = ? AND inserted_at >= ? AND inserted_at < ?", "customer", from, to)
	conversionQuery := p.db.Where("converted_at >= ? AND converted_at < ?", from, to)
	if pageID != "" {
		convQuery = convQuery.Where("page_id = ?", pageID)
		msgQuery = msgQuery.Where("page_id = ?", pageID)
		conversionQuery = conversionQuery.Where("page_id = ?", pageID)
	}

	var conversations []models.PancakeConversation
	var messages []models.PancakeMessage
	var conversions []models.PancakeConversion
	var products []models.Product
	if err := convQuery.Find(&conversations).Error; err != nil ||
		p.db.Where("status = ? OR status IS NULL OR status = ''", "tersedia").Find(&products).Error != nil ||
		msgQuery.Find(&messages).Error != nil || conversionQuery.Find(&conversions).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to calculate analytics"})
		return
	}

	leadIDs := map[string]bool{}
	for _, conversation := range conversations {
		if conversation.FirstCustomerMessageAt != nil && !conversation.FirstCustomerMessageAt.Before(from) && conversation.FirstCustomerMessageAt.Before(to) {
			leadIDs[conversation.ID] = true
		}
	}
	convertedIDs := map[string]bool{}
	for _, conversion := range conversions {
		convertedIDs[conversion.ConversationID] = true
	}
	conversionRate := 0.0
	convertedLeads := 0
	for id := range leadIDs {
		if convertedIDs[id] {
			convertedLeads++
		}
	}
	if len(leadIDs) > 0 {
		conversionRate = float64(convertedLeads) * 100 / float64(len(leadIDs))
	}

	productCounts := countProductMentions(messages, products)
	keywordCounts := countKeywords(messages)
	hourCounts := make([]int, 24)
	for _, message := range messages {
		hourCounts[message.InsertedAt.In(p.loc).Hour()]++
	}
	peakHour, peakCount := 0, 0
	for hour, count := range hourCounts {
		if count > peakCount {
			peakHour, peakCount = hour, count
		}
	}

	allConverted := map[string]bool{}
	allConversionQuery := p.db.Model(&models.PancakeConversion{}).Select("conversation_id")
	if pageID != "" {
		allConversionQuery = allConversionQuery.Where("page_id = ?", pageID)
	}
	var allConversionIDs []string
	_ = allConversionQuery.Pluck("conversation_id", &allConversionIDs).Error
	for _, id := range allConversionIDs {
		allConverted[id] = true
	}
	closing, retarget := scoreCustomers(conversations, allConverted, p.loc)

	type campaignResult struct {
		CampaignID   string `json:"campaign_id"`
		CampaignName string `json:"campaign_name"`
		Sales        int    `json:"sales"`
		Revenue      int64  `json:"revenue"`
	}
	campaignMap := map[string]*campaignResult{}
	for _, conversion := range conversions {
		key := conversion.CampaignID + "|" + conversion.CampaignName
		if conversion.CampaignID == "" && conversion.CampaignName == "" {
			continue
		}
		if campaignMap[key] == nil {
			campaignMap[key] = &campaignResult{CampaignID: conversion.CampaignID, CampaignName: conversion.CampaignName}
		}
		campaignMap[key].Sales++
		campaignMap[key].Revenue += conversion.Amount
	}
	campaigns := make([]campaignResult, 0, len(campaignMap))
	for _, campaign := range campaignMap {
		campaigns = append(campaigns, *campaign)
	}
	sort.Slice(campaigns, func(i, j int) bool { return campaigns[i].Revenue > campaigns[j].Revenue })

	c.JSON(http.StatusOK, gin.H{
		"period":                      gin.H{"from": from, "to": to, "timezone": p.loc.String(), "page_id": pageID},
		"new_leads":                   len(leadIDs),
		"most_asked_products":         topRanked(productCounts, 10),
		"chat_to_purchase":            gin.H{"lead_count": len(leadIDs), "converted_leads": convertedLeads, "rate_percent": conversionRate},
		"closing_potential_customers": closing,
		"retarget_customers":          retarget,
		"wa_campaign_sales":           campaigns,
		"customer_activity":           gin.H{"peak_hour": strconv.Itoa(peakHour) + ":00", "message_count": peakCount, "hourly": hourCounts},
		"top_keywords":                topRanked(keywordCounts, 10),
	})
}

func (p PancakeController) analyticsRange(c *gin.Context) (time.Time, time.Time, bool) {
	now := time.Now().In(p.loc)
	startLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, p.loc)
	from, to := startLocal.UTC(), now.Add(time.Nanosecond).UTC()
	var err error
	if raw := c.Query("from"); raw != "" {
		from, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "from must use RFC3339 format"})
			return time.Time{}, time.Time{}, false
		}
	}
	if raw := c.Query("to"); raw != "" {
		to, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "to must use RFC3339 format"})
			return time.Time{}, time.Time{}, false
		}
	}
	if !from.Before(to) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "from must be before to"})
		return time.Time{}, time.Time{}, false
	}
	return from.UTC(), to.UTC(), true
}

func countProductMentions(messages []models.PancakeMessage, products []models.Product) map[string]int {
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, message := range messages {
		text := strings.ToLower(message.Text)
		for _, product := range products {
			name := strings.TrimSpace(product.NameProduct)
			key := message.ConversationID + "|" + strings.ToLower(name)
			if name != "" && strings.Contains(text, strings.ToLower(name)) && !seen[key] {
				counts[name]++
				seen[key] = true
			}
		}
	}
	return counts
}

var analyticsStopWords = map[string]bool{
	"ada": true, "apa": true, "atau": true, "bisa": true, "buat": true, "dalam": true,
	"dan": true, "dari": true, "dengan": true, "di": true, "ini": true, "itu": true,
	"jadi": true, "juga": true, "ke": true, "kok": true, "lagi": true, "mau": true,
	"nya": true, "saya": true, "sih": true, "sudah": true, "terima": true, "tidak": true,
	"untuk": true, "yang": true, "ya": true, "halo": true, "hai": true, "kak": true,
}

func countKeywords(messages []models.PancakeMessage) map[string]int {
	counts := map[string]int{}
	for _, message := range messages {
		words := strings.FieldsFunc(strings.ToLower(message.Text), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		for _, word := range words {
			if len([]rune(word)) >= 3 && !analyticsStopWords[word] {
				counts[word]++
			}
		}
	}
	return counts
}

func topRanked(counts map[string]int, limit int) []rankedItem {
	items := make([]rankedItem, 0, len(counts))
	for name, count := range counts {
		items = append(items, rankedItem{Name: name, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func scoreCustomers(conversations []models.PancakeConversation, converted map[string]bool, loc *time.Location) ([]customerScore, []customerScore) {
	now := time.Now().In(loc)
	closing := []customerScore{}
	retarget := []customerScore{}
	for _, conversation := range conversations {
		if converted[conversation.ID] || conversation.LastCustomerMessageAt == nil {
			continue
		}
		age := now.Sub(conversation.LastCustomerMessageAt.In(loc))
		if age >= 48*time.Hour && age <= 30*24*time.Hour {
			score := 50
			reasons := []string{"belum membeli", "tidak aktif lebih dari 48 jam"}
			if conversation.HasPhone {
				score += 20
				reasons = append(reasons, "nomor telepon tersedia")
			}
			retarget = append(retarget, customerScore{conversation.ID, conversation.CustomerID, conversation.CustomerName, score, reasons, conversation.LastCustomerMessageAt})
			continue
		}
		if age > 7*24*time.Hour {
			continue
		}
		score := 20
		reasons := []string{"aktif dalam 7 hari terakhir"}
		if conversation.HasPhone {
			score += 25
			reasons = append(reasons, "memberikan nomor telepon")
		}
		if conversation.PageMessageCount > 0 {
			score += 15
			reasons = append(reasons, "sudah ditanggapi tim")
		}
		if conversation.CustomerMessageCount >= 3 {
			score += 20
			reasons = append(reasons, "interaksi tinggi")
		}
		closing = append(closing, customerScore{conversation.ID, conversation.CustomerID, conversation.CustomerName, score, reasons, conversation.LastCustomerMessageAt})
	}
	sort.Slice(closing, func(i, j int) bool { return closing[i].Score > closing[j].Score })
	sort.Slice(retarget, func(i, j int) bool { return retarget[i].Score > retarget[j].Score })
	if len(closing) > 20 {
		closing = closing[:20]
	}
	if len(retarget) > 20 {
		retarget = retarget[:20]
	}
	return closing, retarget
}
