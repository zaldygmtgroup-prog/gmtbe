package models

import "time"

type Notification struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Role      Role       `gorm:"size:30;index;not null" json:"role"`
	Title     string     `gorm:"size:150;not null" json:"title"`
	Message   string     `gorm:"type:text;not null" json:"message"`
	Data      string     `gorm:"type:json" json:"data"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
