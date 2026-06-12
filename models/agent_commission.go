package models

import "time"

type AgentCommission struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"index;not null" json:"user_id"`
	User              *User     `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	ProductName       string    `gorm:"size:150;not null" json:"product_name"`
	ProductPrice      int64     `gorm:"not null" json:"product_price"`
	DiscountAmount    int64     `gorm:"not null;default:0" json:"discount_amount"`
	FinalPrice        int64     `gorm:"not null" json:"final_price"`
	CommissionPercent float64   `gorm:"not null" json:"commission_percent"`
	CommissionAmount  int64     `gorm:"not null" json:"commission_amount"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
