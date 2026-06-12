package models

import "time"

type AgentWallet struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	User             *User     `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	TotalCommission  int64     `gorm:"not null;default:0" json:"total_commission"`
	AvailableBalance int64     `gorm:"not null;default:0" json:"available_balance"`
	PendingWithdraw  int64     `gorm:"not null;default:0" json:"pending_withdraw"`
	WithdrawnBalance int64     `gorm:"not null;default:0" json:"withdrawn_balance"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
