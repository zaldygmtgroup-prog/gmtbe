package controllers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"begmt2/config"
	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AgentController struct {
	cfg config.Config
	db  *gorm.DB
}

func NewAgentController(cfg config.Config, db *gorm.DB) AgentController {
	return AgentController{cfg: cfg, db: db}
}

type calculateCommissionRequest struct {
	ProductName    string `json:"product_name" binding:"required,max=150"`
	ProductPrice   int64  `json:"product_price" binding:"required,min=1"`
	DiscountAmount int64  `json:"discount_amount" binding:"min=0"`
}

type withdrawRequest struct {
	Amount int64 `json:"amount" binding:"required,min=1"`
}

func (a AgentController) GetWallet(c *gin.Context) {
	userID := c.GetUint("user_id")

	wallet, err := a.findOrCreateWallet(a.db, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet": gin.H{
			"total_commission":  wallet.TotalCommission,
			"available_balance": wallet.AvailableBalance,
			"pending_withdraw":  wallet.PendingWithdraw,
			"withdrawn_balance": wallet.WithdrawnBalance,
		},
	})
}

func (a AgentController) CalculateCommission(c *gin.Context) {
	var req calculateCommissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	if req.DiscountAmount >= req.ProductPrice {
		c.JSON(http.StatusBadRequest, gin.H{"message": "discount amount must be lower than product price"})
		return
	}

	userID := c.GetUint("user_id")
	finalPrice := req.ProductPrice - req.DiscountAmount
	commissionAmount := int64(math.Round(float64(finalPrice) * a.cfg.AgentCommissionPercent / 100))

	var commission models.AgentCommission
	var wallet models.AgentWallet
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var err error
		wallet, err = a.findOrCreateWallet(tx, userID)
		if err != nil {
			return err
		}

		commission = models.AgentCommission{
			UserID:            userID,
			ProductName:       req.ProductName,
			ProductPrice:      req.ProductPrice,
			DiscountAmount:    req.DiscountAmount,
			FinalPrice:        finalPrice,
			CommissionPercent: a.cfg.AgentCommissionPercent,
			CommissionAmount:  commissionAmount,
		}
		if err := tx.Create(&commission).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"total_commission":  wallet.TotalCommission + commissionAmount,
			"available_balance": wallet.AvailableBalance + commissionAmount,
		}
		if err := tx.Model(&wallet).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to calculate commission"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "commission calculated",
		"commission": commission,
		"wallet":     wallet,
	})
}

func (a AgentController) CreateWithdraw(c *gin.Context) {
	var req withdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	var withdraw models.WithdrawRequest
	var wallet models.AgentWallet

	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&wallet).Error; err != nil {
			return err
		}

		if wallet.AvailableBalance < req.Amount {
			return errInsufficientBalance
		}

		withdraw = models.WithdrawRequest{
			UserID: userID,
			Amount: req.Amount,
			Status: models.WithdrawStatusOnProgress,
		}
		if err := tx.Create(&withdraw).Error; err != nil {
			return err
		}

		// WD-1003 for ID 12 => 991 + withdraw.ID
		withdraw.WithdrawNumber = fmt.Sprintf("WD-%04d", 991+withdraw.ID)
		if err := tx.Model(&withdraw).Update("withdraw_number", withdraw.WithdrawNumber).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"available_balance": wallet.AvailableBalance - req.Amount,
			"pending_withdraw":  wallet.PendingWithdraw + req.Amount,
		}
		if err := tx.Model(&wallet).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "wallet not found"})
			return
		}
		if errors.Is(err, errInsufficientBalance) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "insufficient wallet balance"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create withdraw request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Withdraw request created",
		"withdraw": gin.H{
			"id":              withdraw.ID,
			"withdraw_number": withdraw.WithdrawNumber,
			"amount":          withdraw.Amount,
			"status":          withdraw.Status,
			"created_at":      withdraw.CreatedAt,
		},
		"wallet": gin.H{
			"total_commission":  wallet.TotalCommission,
			"available_balance": wallet.AvailableBalance,
			"pending_withdraw":  wallet.PendingWithdraw,
			"withdrawn_balance": wallet.WithdrawnBalance,
		},
	})
}

func (a AgentController) ListMyWithdraws(c *gin.Context) {
	userID := c.GetUint("user_id")

	var withdraws []models.WithdrawRequest
	if err := a.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&withdraws).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get withdraw requests"})
		return
	}

	var mappedWithdraws []gin.H
	for _, w := range withdraws {
		mappedWithdraws = append(mappedWithdraws, gin.H{
			"id":              w.ID,
			"withdraw_number": w.WithdrawNumber,
			"amount":          w.Amount,
			"status":          w.Status,
			"created_at":      w.CreatedAt,
			"approved_at":      w.ApprovedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"withdraws": mappedWithdraws})
}

func (a AgentController) ListAllWithdraws(c *gin.Context) {
	status := c.Query("status")

	query := a.db.Preload("User").Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var withdraws []models.WithdrawRequest
	if err := query.Find(&withdraws).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get withdraw requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"withdraws": withdraws})
}

func (a AgentController) ApproveWithdraw(c *gin.Context) {
	withdrawID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid withdraw id"})
		return
	}

	adminID := c.GetUint("user_id")
	now := time.Now()
	var withdraw models.WithdrawRequest
	var wallet models.AgentWallet

	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&withdraw, uint(withdrawID)).Error; err != nil {
			return err
		}

		if withdraw.Status != models.WithdrawStatusOnProgress {
			return errWithdrawAlreadyProcessed
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", withdraw.UserID).
			First(&wallet).Error; err != nil {
			return err
		}

		if wallet.PendingWithdraw < withdraw.Amount {
			return errInvalidWalletBalance
		}

		updates := map[string]interface{}{
			"status":      models.WithdrawStatusApproval,
			"approved_at": &now,
			"approved_by": &adminID,
		}
		if err := tx.Model(&withdraw).Updates(updates).Error; err != nil {
			return err
		}

		walletUpdates := map[string]interface{}{
			"pending_withdraw":  wallet.PendingWithdraw - withdraw.Amount,
			"withdrawn_balance": wallet.WithdrawnBalance + withdraw.Amount,
		}
		if err := tx.Model(&wallet).Updates(walletUpdates).Error; err != nil {
			return err
		}

		withdraw.Status = models.WithdrawStatusApproval
		withdraw.ApprovedAt = &now
		withdraw.ApprovedBy = &adminID
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"message": "withdraw request not found"})
		case errors.Is(err, errWithdrawAlreadyProcessed):
			c.JSON(http.StatusConflict, gin.H{"message": "withdraw request already processed"})
		case errors.Is(err, errInvalidWalletBalance):
			c.JSON(http.StatusConflict, gin.H{"message": "wallet pending balance is invalid"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to approve withdraw request"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "withdraw approved", "withdraw": withdraw, "wallet": wallet})
}

func (a AgentController) findOrCreateWallet(tx *gorm.DB, userID uint) (models.AgentWallet, error) {
	var wallet models.AgentWallet
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&wallet).Error
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return wallet, err
	}

	wallet = models.AgentWallet{UserID: userID}
	if err := tx.Create(&wallet).Error; err != nil {
		return wallet, err
	}

	return wallet, nil
}

var (
	errInsufficientBalance      = errors.New("insufficient balance")
	errWithdrawAlreadyProcessed = errors.New("withdraw already processed")
	errInvalidWalletBalance     = errors.New("invalid wallet balance")
)
