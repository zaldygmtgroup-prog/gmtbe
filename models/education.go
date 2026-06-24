package models

import (
	"time"

	"gorm.io/gorm"
)

type Education struct {
	ID               string         `gorm:"primaryKey;column:id" json:"id"`
	Title            string         `gorm:"column:title" json:"title"`
	Description      string         `gorm:"type:text;column:description" json:"description"`
	FullDescription  string         `gorm:"type:text;column:full_description" json:"full_description"`
	Date             string         `gorm:"column:date" json:"date"`
	Time             string         `gorm:"column:time" json:"time"`
	MaxAttendees     int            `gorm:"column:max_attendees" json:"max_attendees"`
	CurrentAttendees int            `gorm:"column:current_attendees" json:"current_attendees"`
	PriceCurrency    string         `gorm:"column:price_currency" json:"price_currency"`
	PriceAmount      float64        `gorm:"column:price_amount" json:"price_amount"`
	VenueName        string         `gorm:"column:venue_name" json:"venue_name"`
	Room             string         `gorm:"column:room" json:"room"`
	Address          string         `gorm:"column:address" json:"address"`
	City             string         `gorm:"column:city" json:"city"`
	Postcode         string         `gorm:"column:postcode" json:"postcode"`
	Country          string         `gorm:"column:country" json:"country"`
	MapEmbedURL      string         `gorm:"column:map_embed_url" json:"map_embed_url"`
	Type             string         `gorm:"column:type" json:"type"`
	Status           string         `gorm:"column:status" json:"status"` // Open, Closed, Available, Full, Completed
	ThumbnailURL     string         `gorm:"column:thumbnail_url" json:"thumbnail_url"`
	BannerURL        string         `gorm:"column:banner_url" json:"banner_url"`
	CreatedAt        time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`
}
