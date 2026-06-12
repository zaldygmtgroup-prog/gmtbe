package models

import "time"

type WithdrawStatus string

const (
	WithdrawStatusOnProgress WithdrawStatus = "on_progress"
	WithdrawStatusApproval   WithdrawStatus = "approval"
)

type WithdrawRequest struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	WithdrawNumber string         `gorm:"size:50;uniqueIndex;column:withdraw_number" json:"withdraw_number"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	User           *User          `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Amount         int64          `gorm:"not null" json:"amount"`
	Status         WithdrawStatus `gorm:"type:enum('on_progress','approval');default:'on_progress';not null" json:"status"`
	ApprovedAt     *time.Time     `json:"approved_at"`
	ApprovedBy     *uint          `json:"approved_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
