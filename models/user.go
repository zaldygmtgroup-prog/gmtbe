package models

import (
	"time"
)

type Role string

const (
	RoleUser       Role = "user"
	RoleAgent      Role = "agent"
	RoleSuperAdmin Role = "super_admin"
	RoleSales      Role = "sales"
	RoleMarketing  Role = "marketing"
)

type User struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:120;not null" json:"name"`
	TTL         string     `gorm:"size:150;not null" json:"ttl"`
	PhoneNumber string     `gorm:"size:30;not null" json:"phone_number"`
	Gender      string     `gorm:"size:20;not null" json:"gender"`
	Email       string     `gorm:"size:191;uniqueIndex;not null" json:"email"`
	Domicile    string     `gorm:"size:150;not null" json:"domicile"`
	Password    string     `gorm:"size:255;not null" json:"-"`
	Role        Role       `gorm:"type:enum('user','agent','super_admin','sales','marketing');default:'user';not null" json:"role"`
	DetailUser  DetailUser `json:"detail_user,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func IsValidRole(role Role) bool {
	switch role {
	case RoleUser, RoleAgent, RoleSuperAdmin, RoleSales, RoleMarketing:
		return true
	default:
		return false
	}
}
