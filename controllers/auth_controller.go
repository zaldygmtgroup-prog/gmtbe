package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"begmt2/config"
	"begmt2/models"
	"begmt2/services"
	"begmt2/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthController struct {
	cfg            config.Config
	db             *gorm.DB
	pancakeService *services.PancakeService
}

func NewAuthController(cfg config.Config, db *gorm.DB) AuthController {
	return AuthController{
		cfg:            cfg,
		db:             db,
		pancakeService: services.NewPancakeService(cfg),
	}
}

type registerRequest struct {
	Name          string      `json:"name" form:"name"`
	TTL           string      `json:"ttl" form:"ttl"`
	PhoneNumber   string      `json:"phone_number" form:"phone_number"`
	Gender        string      `json:"gender" form:"gender"`
	Email         string      `json:"email" form:"email"`
	Domicile      string      `json:"domicile" form:"domicile"`
	CompanyName   string      `json:"company_name" form:"company_name"`
	Job           *string     `json:"job" form:"job"`
	Instagram     *string     `json:"instagram" form:"instagram"`
	Facebook      *string     `json:"facebook" form:"facebook"`
	Tiktok        *string     `json:"tiktok" form:"tiktok"`
	Photo         *string     `json:"photo" form:"photo"`
	KTPPhoto      *string     `json:"ktp_photo" form:"ktp_photo"`
	FullAddress   *string     `json:"full_address" form:"full_address"`
	BankName      *string     `json:"bank_name" form:"bank_name"`
	AccountNumber *string     `json:"account_number" form:"account_number"`
	Status        *string     `json:"status" form:"status"`
	Password      string      `json:"password" form:"password"`
	Role          models.Role `json:"role" form:"role"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Client   string `json:"client" binding:"omitempty,max=100"`
}

type googleAuthRequest struct {
	IDToken string `json:"id_token" binding:"required"`
	Client  string `json:"client" binding:"omitempty,max=100"`
}

type googleTokenInfo struct {
	Audience      string      `json:"aud"`
	Email         string      `json:"email"`
	EmailVerified interface{} `json:"email_verified"`
	Name          string      `json:"name"`
	Picture       string      `json:"picture"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type verifyResetTokenRequest struct {
	Email string `json:"email" binding:"required,email"`
	Token string `json:"token" binding:"required,len=6"`
}

type resetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Token       string `json:"token" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type applyAgentRequest struct {
	Job              *string `json:"job"`
	Instagram        *string `json:"instagram"`
	Facebook         *string `json:"facebook"`
	Tiktok           *string `json:"tiktok"`
	AgentProgramType *string `json:"agent_program_type"`
	AgentMotivation  *string `json:"agent_motivation"`
	ReferralSource   *string `json:"referral_source"`
	ReferralName     *string `json:"referral_name"`
	ReferralOther    *string `json:"referral_other"`
	TargetProduct    *string `json:"target_product"`
}

type completeAgentVerificationRequest struct {
	Photo         string `json:"photo" form:"photo"`
	KTPPhoto      string `json:"ktp_photo" form:"ktp_photo"`
	BankName      string `json:"bank_name" form:"bank_name" binding:"required"`
	AccountNumber string `json:"account_number" form:"account_number" binding:"required"`
	TTL           string `json:"ttl" form:"ttl" binding:"required"`
	FullAddress   string `json:"full_address" form:"full_address" binding:"required"`
	Domicile      string `json:"domicile" form:"domicile"`
}

type updateAgentApplicationStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type createSSOCodeRequest struct {
	TargetClient string  `json:"target_client" binding:"required,max=100"`
	RedirectURI  *string `json:"redirect_uri" binding:"omitempty,max=255"`
	State        *string `json:"state" binding:"omitempty,max=255"`
}

type exchangeSSOCodeRequest struct {
	Code         string `json:"code" binding:"required"`
	TargetClient string `json:"target_client" binding:"required,max=100"`
}

func (a AuthController) Register(c *gin.Context) {
	var req registerRequest
	if err := bindRegisterRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	if isMultipartRequest(c) {
		if photo, ok, err := saveOptionalImageUpload(c, a.cfg.UploadDir, "photo", "users"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid photo", "error": err.Error()})
			return
		} else if ok {
			req.Photo = &photo
		}
		if ktpPhoto, ok, err := saveOptionalImageUpload(c, a.cfg.UploadDir, "ktp_photo", "ktp"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid ktp_photo", "error": err.Error()})
			return
		} else if ok {
			req.KTPPhoto = &ktpPhoto
		}
	}

	if req.Role == "" || !models.IsValidRole(req.Role) {
		req.Role = models.RoleUser
	}

	var existingUser models.User
	if err := a.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"message": "email already registered"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to check email"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to hash password"})
		return
	}

	user := models.User{
		Name:        req.Name,
		TTL:         req.TTL,
		PhoneNumber: req.PhoneNumber,
		Gender:      req.Gender,
		Email:       req.Email,
		Domicile:    req.Domicile,
		Password:    hashedPassword,
		Role:        req.Role,
	}

	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		detailUser := models.DetailUser{
			UserID:        user.ID,
			CompanyName:   req.CompanyName,
			Job:           req.Job,
			Instagram:     req.Instagram,
			Facebook:      req.Facebook,
			Tiktok:        req.Tiktok,
			Photo:         req.Photo,
			KTPPhoto:      req.KTPPhoto,
			FullAddress:   req.FullAddress,
			BankName:      req.BankName,
			AccountNumber: req.AccountNumber,
			Status:        req.Status,
		}

		if err := tx.Create(&detailUser).Error; err != nil {
			return err
		}

		user.DetailUser = detailUser
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "registration successful", "user": user})
}

func bindRegisterRequest(c *gin.Context, req *registerRequest) error {
	if isMultipartRequest(c) {
		req.Name = c.PostForm("name")
		req.TTL = c.PostForm("ttl")
		req.PhoneNumber = c.PostForm("phone_number")
		req.Gender = c.PostForm("gender")
		req.Email = c.PostForm("email")
		req.Domicile = c.PostForm("domicile")
		req.CompanyName = c.PostForm("company_name")
		req.Job = optionalFormString(c, "job")
		req.Instagram = optionalFormString(c, "instagram")
		req.Facebook = optionalFormString(c, "facebook")
		req.Tiktok = optionalFormString(c, "tiktok")
		req.Photo = optionalFormString(c, "photo")
		req.KTPPhoto = optionalFormString(c, "ktp_photo")
		req.FullAddress = optionalFormString(c, "full_address")
		req.BankName = optionalFormString(c, "bank_name")
		req.AccountNumber = optionalFormString(c, "account_number")
		req.Status = optionalFormString(c, "status")
		req.Password = c.PostForm("password")
		req.Role = models.Role(c.PostForm("role"))
		return nil
	}
	return c.ShouldBindJSON(req)
}

func (a AuthController) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	var user models.User
	if err := a.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "email or password is incorrect"})
		return
	}

	if !utils.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "email or password is incorrect"})
		return
	}

	token, session, err := a.issueToken(user, req.Client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "login successful", "token": token, "session": session, "user": user})
}

func (a AuthController) LoginWithGoogle(c *gin.Context) {
	var req googleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	if a.cfg.GoogleClientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "google login is not configured"})
		return
	}

	profile, err := a.verifyGoogleIDToken(req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid google token", "error": err.Error()})
		return
	}

	var user models.User
	err = a.db.Transaction(func(tx *gorm.DB) error {
		email := strings.ToLower(strings.TrimSpace(profile.Email))
		if err := tx.Preload("DetailUser").Where("email = ?", email).First(&user).Error; err == nil {
			if user.DetailUser.ID == 0 {
				detail := models.DetailUser{
					UserID:      user.ID,
					CompanyName: "-",
				}
				if err := tx.Create(&detail).Error; err != nil {
					return err
				}
				user.DetailUser = detail
			}
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		createdUser, err := createGoogleUser(profile)
		if err != nil {
			return err
		}
		if err := tx.Create(&createdUser).Error; err != nil {
			return err
		}

		user = createdUser
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to login with google"})
		return
	}

	token, session, err := a.issueToken(user, req.Client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "google login successful", "token": token, "session": session, "user": user})
}

func (a AuthController) RegisterWithGoogle(c *gin.Context) {
	var req googleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	if a.cfg.GoogleClientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "google register is not configured"})
		return
	}

	profile, err := a.verifyGoogleIDToken(req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid google token", "error": err.Error()})
		return
	}

	var user models.User
	err = a.db.Transaction(func(tx *gorm.DB) error {
		email := strings.ToLower(strings.TrimSpace(profile.Email))
		var existingUser models.User
		if err := tx.Where("email = ?", email).First(&existingUser).Error; err == nil {
			return errGoogleEmailAlreadyRegistered
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		createdUser, err := createGoogleUser(profile)
		if err != nil {
			return err
		}
		if err := tx.Create(&createdUser).Error; err != nil {
			return err
		}

		user = createdUser
		return nil
	})
	if err != nil {
		if errors.Is(err, errGoogleEmailAlreadyRegistered) {
			c.JSON(http.StatusConflict, gin.H{"message": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to register with google"})
		return
	}

	token, session, err := a.issueToken(user, req.Client)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "google registration successful", "token": token, "session": session, "user": user})
}

func (a AuthController) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	var user models.User
	if err := a.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "email not registered"})
		return
	}

	token, err := utils.GenerateResetToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate reset token"})
		return
	}

	now := time.Now()
	if err := a.db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL AND expires_at > ?", user.ID, now).
		Update("used_at", now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to invalidate old reset tokens"})
		return
	}

	resetToken := models.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: utils.HashToken(token),
		ExpiresAt: now.Add(time.Duration(a.cfg.ResetTokenExpiresMinutes) * time.Minute),
	}

	if err := a.db.Create(&resetToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save reset token"})
		return
	}

	if err := a.pancakeService.SendPasswordResetToken(user.PhoneNumber, user.Name, token, a.cfg.ResetTokenExpiresMinutes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to send reset token whatsapp", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reset token sent to whatsapp"})
}

func (a AuthController) VerifyResetToken(c *gin.Context) {
	var req verifyResetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	if _, ok := a.findValidResetToken(req.Email, req.Token); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid or expired reset token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reset token is valid"})
}

func (a AuthController) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	resetToken, ok := a.findValidResetToken(req.Email, req.Token)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid or expired reset token"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to hash password"})
		return
	}

	now := time.Now()
	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", resetToken.UserID).Update("password", hashedPassword).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.PasswordResetToken{}).Where("id = ?", resetToken.ID).Update("used_at", now).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password reset successful"})
}

func (a AuthController) Me(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user models.User
	if err := a.db.Preload("DetailUser").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (a AuthController) Session(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.GetString("session_id")

	var user models.User
	if err := a.db.Preload("DetailUser").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"session_id":    sessionID,
		"user":          user,
	})
}

func (a AuthController) Logout(c *gin.Context) {
	userID := c.GetUint("user_id")
	now := time.Now()

	if err := a.db.Model(&models.AuthSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logout successful"})
}

func (a AuthController) CreateSSOCode(c *gin.Context) {
	var req createSSOCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	redirectURI, ok := a.resolveSSORedirect(req.TargetClient, req.RedirectURI)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "target client or redirect uri is not allowed"})
		return
	}

	code, err := utils.GenerateOpaqueToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate sso code"})
		return
	}

	ssoCode := models.SSOCode{
		CodeHash:     utils.HashToken(code),
		UserID:       c.GetUint("user_id"),
		TargetClient: req.TargetClient,
		RedirectURI:  redirectURI,
		ExpiresAt:    time.Now().Add(time.Duration(a.cfg.SSOCodeExpiresSeconds) * time.Second),
	}
	if err := a.db.Create(&ssoCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save sso code"})
		return
	}

	redirectURL, err := buildSSORedirectURL(redirectURI, code, req.State)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to build redirect url"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":         code,
		"expires_at":   ssoCode.ExpiresAt,
		"redirect_url": redirectURL,
	})
}

func (a AuthController) ExchangeSSOCode(c *gin.Context) {
	var req exchangeSSOCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	var user models.User
	var session models.AuthSession
	var token string
	now := time.Now()

	err := a.db.Transaction(func(tx *gorm.DB) error {
		var ssoCode models.SSOCode
		err := tx.Where(
			"code_hash = ? AND target_client = ? AND used_at IS NULL AND expires_at > ?",
			utils.HashToken(req.Code),
			req.TargetClient,
			now,
		).First(&ssoCode).Error
		if err != nil {
			return err
		}

		if _, ok := a.cfg.SSOClientRedirects[ssoCode.TargetClient]; !ok {
			return errors.New("sso client is not configured")
		}

		if err := tx.Model(&ssoCode).Update("used_at", &now).Error; err != nil {
			return err
		}

		if err := tx.Preload("DetailUser").First(&user, ssoCode.UserID).Error; err != nil {
			return err
		}

		var sessionErr error
		session, sessionErr = a.createSession(tx, user.ID, req.TargetClient)
		if sessionErr != nil {
			return sessionErr
		}

		token, sessionErr = utils.GenerateJWT(user.ID, user.Email, string(user.Role), session.SessionID, a.cfg.JWTSecret, a.cfg.JWTExpiresHours)
		return sessionErr
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid or expired sso code"})
			return
		}
		if err.Error() == "sso client is not configured" {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to exchange sso code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "sso exchange successful",
		"token":   token,
		"session": session,
		"user":    user,
	})
}

func (a AuthController) ApplyAgent(c *gin.Context) {
	var req applyAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	status := "verif"

	var user models.User
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("DetailUser").First(&user, userID).Error; err != nil {
			return err
		}

		if user.Role != models.RoleUser {
			return gorm.ErrInvalidData
		}

		detailUser := user.DetailUser
		detailUser.UserID = user.ID
		detailUser.Job = req.Job
		detailUser.Instagram = req.Instagram
		detailUser.Facebook = req.Facebook
		detailUser.Tiktok = req.Tiktok
		detailUser.AgentProgramType = req.AgentProgramType
		detailUser.AgentMotivation = req.AgentMotivation
		detailUser.ReferralSource = req.ReferralSource
		detailUser.ReferralName = req.ReferralName
		detailUser.ReferralOther = req.ReferralOther
		detailUser.TargetProduct = req.TargetProduct
		detailUser.Status = &status

		if detailUser.ID == 0 {
			if detailUser.CompanyName == "" {
				detailUser.CompanyName = "-"
			}
			if err := tx.Create(&detailUser).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&detailUser).Error; err != nil {
			return err
		}

		user.DetailUser = detailUser
		return nil
	})
	if err != nil {
		if err == gorm.ErrInvalidData {
			c.JSON(http.StatusConflict, gin.H{"message": "only regular users can apply to become an agent"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to submit agent application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agent application submitted", "user": user})
}

func (a AuthController) CompleteAgentVerification(c *gin.Context) {
	var req completeAgentVerificationRequest
	if err := bindCompleteAgentVerificationRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	if isMultipartRequest(c) {
		photo, err := saveRequiredImageUpload(c, a.cfg.UploadDir, "photo", "users")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid photo", "error": err.Error()})
			return
		}
		ktpPhoto, err := saveRequiredImageUpload(c, a.cfg.UploadDir, "ktp_photo", "ktp")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid ktp_photo", "error": err.Error()})
			return
		}
		req.Photo = photo
		req.KTPPhoto = ktpPhoto
	}
	if req.Photo == "" || req.KTPPhoto == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "photo and ktp_photo are required"})
		return
	}

	userID := c.GetUint("user_id")
	var user models.User

	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("DetailUser").First(&user, userID).Error; err != nil {
			return err
		}

		if user.DetailUser.ID == 0 || user.DetailUser.Status == nil || *user.DetailUser.Status != "verif" {
			return errAgentMustBeVerified
		}

		if err := tx.Model(&user).Updates(map[string]interface{}{
			"ttl":      req.TTL,
			"domicile": req.Domicile,
		}).Error; err != nil {
			return err
		}
		user.TTL = req.TTL
		user.Domicile = req.Domicile

		user.DetailUser.Photo = &req.Photo
		user.DetailUser.KTPPhoto = &req.KTPPhoto
		user.DetailUser.BankName = &req.BankName
		user.DetailUser.AccountNumber = &req.AccountNumber
		user.DetailUser.FullAddress = &req.FullAddress

		return tx.Save(&user.DetailUser).Error
	})
	if err != nil {
		if errors.Is(err, errAgentMustBeVerified) {
			c.JSON(http.StatusConflict, gin.H{"message": "agent application must be verified by admin before completing verification data"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to complete agent verification", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification data completed", "user": user})
}

func bindCompleteAgentVerificationRequest(c *gin.Context, req *completeAgentVerificationRequest) error {
	if isMultipartRequest(c) {
		req.BankName = c.PostForm("bank_name")
		req.AccountNumber = c.PostForm("account_number")
		req.TTL = c.PostForm("ttl")
		req.FullAddress = c.PostForm("full_address")
		req.Domicile = c.PostForm("domicile")
		if req.BankName == "" {
			return errors.New("bank_name is required")
		}
		if req.AccountNumber == "" {
			return errors.New("account_number is required")
		}
		if req.TTL == "" {
			return errors.New("ttl is required")
		}
		if req.FullAddress == "" {
			return errors.New("full_address is required")
		}
		return nil
	}
	return c.ShouldBindJSON(req)
}

func optionalFormString(c *gin.Context, key string) *string {
	value := c.PostForm(key)
	if value == "" {
		return nil
	}
	return &value
}

func (a AuthController) ListAgentApplications(c *gin.Context) {
	status := c.Query("status")

	query := a.db.Preload("DetailUser").
		Joins("JOIN detail_users ON detail_users.user_id = users.id").
		Where("detail_users.status IS NOT NULL").
		Order("detail_users.updated_at DESC")
	if status != "" {
		query = query.Where("detail_users.status = ?", status)
	}

	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get agent applications", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"applications": users})
}

func (a AuthController) UpdateAgentApplicationStatus(c *gin.Context) {
	var req updateAgentApplicationStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	if !isValidAgentApplicationStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid agent application status"})
		return
	}

	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid user id"})
		return
	}

	var user models.User
	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("DetailUser").First(&user, uint(userID)).Error; err != nil {
			return err
		}
		if user.DetailUser.ID == 0 {
			return errAgentApplicationNotFound
		}

		updates := map[string]interface{}{"status": req.Status}
		if err := tx.Model(&user.DetailUser).Updates(updates).Error; err != nil {
			return err
		}
		user.DetailUser.Status = &req.Status

		nextRole := user.Role
		if req.Status == "official_agent" {
			nextRole = models.RoleAgent
		} else if user.Role == models.RoleAgent && req.Status != "official_agent" {
			nextRole = models.RoleUser
		}
		if nextRole != user.Role {
			if err := tx.Model(&user).Update("role", nextRole).Error; err != nil {
				return err
			}
			user.Role = nextRole
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, errAgentApplicationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "agent application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update agent application status", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agent application status updated", "user": user})
}

func (a AuthController) issueToken(user models.User, client string) (string, models.AuthSession, error) {
	session, err := a.createSession(a.db, user.ID, client)
	if err != nil {
		return "", models.AuthSession{}, err
	}

	token, err := utils.GenerateJWT(user.ID, user.Email, string(user.Role), session.SessionID, a.cfg.JWTSecret, a.cfg.JWTExpiresHours)
	if err != nil {
		return "", models.AuthSession{}, err
	}

	return token, session, nil
}

func (a AuthController) createSession(tx *gorm.DB, userID uint, client string) (models.AuthSession, error) {
	sessionID, err := utils.GenerateOpaqueToken(32)
	if err != nil {
		return models.AuthSession{}, err
	}

	session := models.AuthSession{
		SessionID: sessionID,
		UserID:    userID,
		Client:    client,
		ExpiresAt: time.Now().Add(time.Duration(a.cfg.JWTExpiresHours) * time.Hour),
	}
	if err := tx.Create(&session).Error; err != nil {
		return models.AuthSession{}, err
	}

	return session, nil
}

func createGoogleUser(profile googleTokenInfo) (models.User, error) {
	email := strings.ToLower(strings.TrimSpace(profile.Email))
	randomPassword, err := utils.GenerateOpaqueToken(32)
	if err != nil {
		return models.User{}, err
	}
	hashedPassword, err := utils.HashPassword(randomPassword)
	if err != nil {
		return models.User{}, err
	}

	name := strings.TrimSpace(profile.Name)
	if name == "" {
		name = email
	}

	user := models.User{
		Name:        name,
		TTL:         "-",
		PhoneNumber: "-",
		Gender:      "-",
		Email:       email,
		Domicile:    "-",
		Password:    hashedPassword,
		Role:        models.RoleUser,
		DetailUser: models.DetailUser{
			CompanyName: "-",
		},
	}
	if profile.Picture != "" {
		user.DetailUser.Photo = &profile.Picture
	}

	return user, nil
}

func (a AuthController) verifyGoogleIDToken(idToken string) (googleTokenInfo, error) {
	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	client := http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(endpoint)
	if err != nil {
		return googleTokenInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return googleTokenInfo{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return googleTokenInfo{}, fmt.Errorf("google tokeninfo returned status %d", resp.StatusCode)
	}

	var profile googleTokenInfo
	if err := json.Unmarshal(body, &profile); err != nil {
		return googleTokenInfo{}, err
	}
	if profile.Audience != a.cfg.GoogleClientID {
		return googleTokenInfo{}, errors.New("google token audience does not match")
	}
	if strings.TrimSpace(profile.Email) == "" {
		return googleTokenInfo{}, errors.New("google token does not contain email")
	}
	if !isGoogleEmailVerified(profile.EmailVerified) {
		return googleTokenInfo{}, errors.New("google email is not verified")
	}

	return profile, nil
}

func isGoogleEmailVerified(value interface{}) bool {
	switch verified := value.(type) {
	case bool:
		return verified
	case string:
		return strings.EqualFold(verified, "true")
	default:
		return false
	}
}

func (a AuthController) resolveSSORedirect(targetClient string, requestedRedirectURI *string) (string, bool) {
	configuredRedirectURI, ok := a.cfg.SSOClientRedirects[targetClient]
	if !ok || configuredRedirectURI == "" {
		return "", false
	}

	if requestedRedirectURI != nil && *requestedRedirectURI != configuredRedirectURI {
		return "", false
	}

	return configuredRedirectURI, true
}

func buildSSORedirectURL(redirectURI, code string, state *string) (string, error) {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("code", code)
	if state != nil && *state != "" {
		query.Set("state", *state)
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func isValidAgentApplicationStatus(status string) bool {
	switch status {
	case "not_verif", "verif", "official_agent", "stopped_agent":
		return true
	default:
		return false
	}
}

func (a AuthController) findValidResetToken(email, token string) (models.PasswordResetToken, bool) {
	var user models.User
	if err := a.db.Where("email = ?", email).First(&user).Error; err != nil {
		return models.PasswordResetToken{}, false
	}

	var resetToken models.PasswordResetToken
	err := a.db.Where(
		"user_id = ? AND token_hash = ? AND used_at IS NULL AND expires_at > ?",
		user.ID,
		utils.HashToken(token),
		time.Now(),
	).First(&resetToken).Error

	return resetToken, err == nil
}

var (
	errAgentMustBeVerified          = errors.New("agent must be verified")
	errAgentApplicationNotFound     = errors.New("agent application not found")
	errGoogleEmailAlreadyRegistered = errors.New("google email already registered")
)
