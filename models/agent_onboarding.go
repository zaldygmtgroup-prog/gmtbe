package models

import "time"

type OnboardingStatus string

const (
	OnboardingStatusNotStarted OnboardingStatus = "not_started"
	OnboardingStatusInProgress OnboardingStatus = "in_progress"
	OnboardingStatusCompleted  OnboardingStatus = "completed"
)

type AgentOnboardingVideo struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Slug            string    `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Title           string    `gorm:"size:255;not null" json:"title"`
	Description     string    `gorm:"type:text" json:"description"`
	VideoURL        string    `gorm:"size:255;not null;column:video_url" json:"video_url"`
	DurationSeconds int       `gorm:"not null" json:"duration_seconds"`
	SortOrder       int       `gorm:"not null;default:0" json:"sort_order"`
	IsRequired      bool      `gorm:"not null;default:true" json:"is_required"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AgentOnboardingProgress struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	UserID         uint             `gorm:"uniqueIndex:idx_user_video;not null" json:"user_id"`
	User           *User            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	VideoID        uint             `gorm:"uniqueIndex:idx_user_video;not null" json:"video_id"`
	Video          *AgentOnboardingVideo `gorm:"foreignKey:VideoID;constraint:OnDelete:CASCADE" json:"video,omitempty"`
	Status         OnboardingStatus `gorm:"type:enum('not_started','in_progress','completed');default:'not_started';not null" json:"status"`
	WatchedSeconds int              `gorm:"not null;default:0" json:"watched_seconds"`
	CompletedAt    *time.Time       `json:"completed_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}
