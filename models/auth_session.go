package models

import "time"

type AuthSession struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	SessionID        string     `gorm:"size:191;uniqueIndex;not null" json:"session_id"`
	UserID           uint       `gorm:"index;not null" json:"user_id"`
	User             User       `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Client           string     `gorm:"size:100" json:"client"`
	RefreshTokenHash *string    `gorm:"size:64;index" json:"-"`
	ExpiresAt        time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
