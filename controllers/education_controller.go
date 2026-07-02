package controllers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"begmt2/config"
	"begmt2/models"
	"begmt2/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EducationController struct {
	DB  *gorm.DB
	Cfg config.Config
}

type educationResponse struct {
	models.Education
	IsRegistered bool `json:"is_registered"`
}

func NewEducationController(cfg config.Config, db *gorm.DB) *EducationController {
	return &EducationController{
		DB:  db,
		Cfg: cfg,
	}
}

// ListEducations GET /api/educations
func (c *EducationController) ListEducations(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if limit < 1 {
		limit = 10
	}

	month := ctx.Query("month")
	eventType := ctx.Query("type")
	status := ctx.Query("status")

	query := c.DB.Model(&models.Education{})

	if month != "" {
		query = query.Where("date LIKE ?", month+"%")
	}
	if eventType != "" {
		query = query.Where("type = ?", eventType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var educations []models.Education
	if err := query.Offset(offset).Limit(limit).Find(&educations).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to retrieve educations"})
		return
	}

	data := c.buildEducationListResponse(ctx, educations)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "List of education events retrieved successfully",
		"data":    data,
		"meta": gin.H{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

// GetEducation GET /api/educations/:id
func (c *EducationController) GetEducation(ctx *gin.Context) {
	id := ctx.Param("id")

	var education models.Education
	if err := c.DB.Where("id = ?", id).First(&education).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Education event not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to retrieve education event"})
		return
	}

	isRegistered := c.isEducationRegistered(ctx, education.ID)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": educationResponse{
			Education:    education,
			IsRegistered: isRegistered,
		},
	})
}

// CreateEducation POST /api/educations (CRUD)
func (c *EducationController) CreateEducation(ctx *gin.Context) {
	var input models.Education
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	input.ID = "edu_" + uuid.New().String()
	if input.Status == "" {
		input.Status = "Available"
	}

	if err := c.DB.Create(&input).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create education event"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Education event created successfully",
		"data":    input,
	})
}

// UpdateEducation PUT /api/educations/:id (CRUD)
func (c *EducationController) UpdateEducation(ctx *gin.Context) {
	id := ctx.Param("id")

	var education models.Education
	if err := c.DB.Where("id = ?", id).First(&education).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Education event not found"})
		return
	}

	var input models.Education
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := c.DB.Model(&education).Updates(input).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update education event"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Education event updated successfully",
		"data":    education,
	})
}

// DeleteEducation DELETE /api/educations/:id (CRUD)
func (c *EducationController) DeleteEducation(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.DB.Where("id = ?", id).Delete(&models.Education{}).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to delete education event"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Education event deleted successfully",
	})
}

// RegisterEducation POST /api/educations/:id/register
func (c *EducationController) RegisterEducation(ctx *gin.Context) {
	eventID := ctx.Param("id")
	userID := ctx.GetUint("user_id")
	if userID == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	var input struct {
		Salutation    string `json:"salutation" binding:"required"`
		FirstName     string `json:"first_name" binding:"required"`
		Surname       string `json:"surname" binding:"required"`
		Email         string `json:"email" binding:"required,email"`
		ConfirmEmail  string `json:"confirm_email" binding:"required,email"`
		PhoneLandline string `json:"phone_landline"`
		PhoneMobile   string `json:"phone_mobile" binding:"required"`
		Company       string `json:"company"`
		Position      string `json:"position"`
		Address       struct {
			Street   string `json:"street" binding:"required"`
			Postcode string `json:"postcode" binding:"required"`
			Town     string `json:"town" binding:"required"`
			Country  string `json:"country" binding:"required"`
		} `json:"address"`
		MealPreference        string `json:"meal_preference"`
		AdditionalInformation string `json:"additional_information"`
		Consents              struct {
			ConditionsOfParticipation bool `json:"conditions_of_participation"`
			PrivacyPolicy             bool `json:"privacy_policy"`
			MarketingUpdates          bool `json:"marketing_updates"`
		} `json:"consents"`
		RecaptchaToken string `json:"recaptcha_token"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Validation failed", "errors": err.Error()})
		return
	}

	errors := make(map[string]string)
	if input.Email != input.ConfirmEmail {
		errors["email"] = "Email and Confirm Email do not match"
	}
	if !input.Consents.PrivacyPolicy {
		errors["consents.privacy_policy"] = "You must accept the privacy policy"
	}
	if !input.Consents.ConditionsOfParticipation {
		errors["consents.conditions_of_participation"] = "You must accept the conditions of participation"
	}

	if len(errors) > 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Validation failed", "errors": errors})
		return
	}

	var education models.Education
	if err := c.DB.Where("id = ?", eventID).First(&education).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Education event not found"})
		return
	}

	if education.MaxAttendees > 0 && education.CurrentAttendees >= education.MaxAttendees {
		ctx.JSON(http.StatusConflict, gin.H{"success": false, "message": "Event is already fully booked."})
		return
	}

	// Check if already registered
	var existingReg models.EducationRegistration
	if err := c.DB.Where("event_id = ? AND user_id = ?", eventID, userID).First(&existingReg).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{"success": false, "message": "You are already registered for this event."})
		return
	}

	registration := models.EducationRegistration{
		ID:                        "reg_" + uuid.New().String(),
		EventID:                   eventID,
		UserID:                    userID,
		Salutation:                input.Salutation,
		FirstName:                 input.FirstName,
		Surname:                   input.Surname,
		Email:                     input.Email,
		PhoneLandline:             input.PhoneLandline,
		PhoneMobile:               input.PhoneMobile,
		Company:                   input.Company,
		Position:                  input.Position,
		Street:                    input.Address.Street,
		Postcode:                  input.Address.Postcode,
		Town:                      input.Address.Town,
		Country:                   input.Address.Country,
		MealPreference:            input.MealPreference,
		AdditionalInformation:     input.AdditionalInformation,
		ConditionsOfParticipation: input.Consents.ConditionsOfParticipation,
		PrivacyPolicy:             input.Consents.PrivacyPolicy,
		MarketingUpdates:          input.Consents.MarketingUpdates,
		Status:                    "Confirmed", // default to Confirmed
		CreatedAt:                 time.Now(),
		UpdatedAt:                 time.Now(),
	}

	err := c.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&registration).Error; err != nil {
			return err
		}

		if err := tx.Model(&education).Update("current_attendees", gorm.Expr("current_attendees + ?", 1)).Error; err != nil {
			return err
		}

		// Optional: if max attendees reached, set status to Full
		if education.MaxAttendees > 0 && (education.CurrentAttendees+1) >= education.MaxAttendees {
			if err := tx.Model(&education).Update("status", "Full").Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to register for the event"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Registration successful. Check your email for the ticket.",
		"data": gin.H{
			"registration_id": registration.ID,
			"event_id":        registration.EventID,
			"user_id":         fmt.Sprintf("usr_%d", registration.UserID), // Mock user ID format matching response
			"status":          registration.Status,
		},
	})
}

func (c *EducationController) buildEducationListResponse(ctx *gin.Context, educations []models.Education) []educationResponse {
	response := make([]educationResponse, 0, len(educations))
	for _, education := range educations {
		response = append(response, educationResponse{Education: education})
	}

	userID, ok := c.optionalAuthenticatedUserID(ctx)
	if !ok || len(educations) == 0 {
		return response
	}

	eventIDs := make([]string, 0, len(educations))
	for _, education := range educations {
		eventIDs = append(eventIDs, education.ID)
	}

	var registrations []models.EducationRegistration
	if err := c.DB.
		Select("event_id").
		Where("user_id = ? AND event_id IN ?", userID, eventIDs).
		Find(&registrations).Error; err != nil {
		return response
	}

	registeredEvents := make(map[string]bool, len(registrations))
	for _, registration := range registrations {
		registeredEvents[registration.EventID] = true
	}

	for i := range response {
		response[i].IsRegistered = registeredEvents[response[i].ID]
	}

	return response
}

func (c *EducationController) isEducationRegistered(ctx *gin.Context, eventID string) bool {
	userID, ok := c.optionalAuthenticatedUserID(ctx)
	if !ok {
		return false
	}

	var count int64
	if err := c.DB.Model(&models.EducationRegistration{}).
		Where("event_id = ? AND user_id = ?", eventID, userID).
		Count(&count).Error; err != nil {
		return false
	}

	return count > 0
}

func (c *EducationController) optionalAuthenticatedUserID(ctx *gin.Context) (uint, bool) {
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		return 0, false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0, false
	}

	claims, err := utils.ParseJWT(parts[1], c.Cfg.JWTSecret)
	if err != nil || claims.SessionID == "" {
		return 0, false
	}

	var session models.AuthSession
	err = c.DB.Where(
		"session_id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?",
		claims.SessionID,
		claims.UserID,
		time.Now(),
	).First(&session).Error
	if err != nil {
		return 0, false
	}

	return claims.UserID, true
}
