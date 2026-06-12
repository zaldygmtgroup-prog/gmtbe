package models

import "time"

type SSOCode struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	CodeHash     string     `gorm:"size:255;uniqueIndex;not null" json:"-"`
	UserID       uint       `gorm:"index;not null" json:"user_id"`
	User         User       `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	TargetClient string     `gorm:"size:100;index;not null" json:"target_client"`
	RedirectURI  string     `gorm:"size:255;not null" json:"redirect_uri"`
	ExpiresAt    time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt       *time.Time `json:"used_at"`
	CreatedAt    time.Time  `json:"created_at"`
}
