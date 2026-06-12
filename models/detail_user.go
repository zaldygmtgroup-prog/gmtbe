package models

import "time"

type DetailUser struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	User             *User     `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	CompanyName      string    `gorm:"size:150;not null" json:"company_name"`
	Job              *string   `gorm:"size:120" json:"job"`
	Instagram        *string   `gorm:"size:120" json:"instagram"`
	Facebook         *string   `gorm:"size:120" json:"facebook"`
	Tiktok           *string   `gorm:"size:120" json:"tiktok"`
	AgentProgramType *string   `gorm:"size:50" json:"agent_program_type"`
	AgentMotivation  *string   `gorm:"type:text" json:"agent_motivation"`
	ReferralSource   *string   `gorm:"size:80" json:"referral_source"`
	ReferralName     *string   `gorm:"size:120" json:"referral_name"`
	ReferralOther    *string   `gorm:"size:255" json:"referral_other"`
	TargetProduct    *string   `gorm:"size:255" json:"target_product"`
	Photo            *string   `gorm:"size:255" json:"photo"`
	KTPPhoto         *string   `gorm:"size:255" json:"ktp_photo"`
	FullAddress      *string   `gorm:"type:text" json:"full_address"`
	BankName         *string   `gorm:"size:120" json:"bank_name"`
	AccountNumber    *string   `gorm:"size:80" json:"account_number"`
	Status           *string   `gorm:"size:50" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
