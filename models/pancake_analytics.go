package models

import "time"

// PancakeConversation is a compact, query-friendly projection of a Pancake chat.
// The raw conversation can always be reconstructed from Pancake; this table keeps
// only the fields needed by the analytics API.
type PancakeConversation struct {
	ID                     string     `gorm:"primaryKey;size:191" json:"id"`
	PageID                 string     `gorm:"size:191;index;not null" json:"page_id"`
	CustomerID             string     `gorm:"size:191;index;not null" json:"customer_id"`
	PageCustomerID         string     `gorm:"size:191;index" json:"page_customer_id,omitempty"`
	CustomerName           string     `gorm:"size:255" json:"customer_name"`
	Type                   string     `gorm:"size:30;index" json:"type"`
	PostID                 string     `gorm:"size:191;index" json:"post_id,omitempty"`
	PostMessage            string     `gorm:"type:text" json:"post_message,omitempty"`
	FirstCustomerMessageAt *time.Time `gorm:"index" json:"first_customer_message_at,omitempty"`
	LastCustomerMessageAt  *time.Time `gorm:"index" json:"last_customer_message_at,omitempty"`
	LastPageMessageAt      *time.Time `gorm:"index" json:"last_page_message_at,omitempty"`
	CustomerMessageCount   int        `gorm:"not null;default:0" json:"customer_message_count"`
	PageMessageCount       int        `gorm:"not null;default:0" json:"page_message_count"`
	HasPhone               bool       `gorm:"not null;default:false" json:"has_phone"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type PancakeMessage struct {
	ID             string    `gorm:"primaryKey;size:191" json:"id"`
	PageID         string    `gorm:"size:191;index;not null" json:"page_id"`
	ConversationID string    `gorm:"size:191;index;not null" json:"conversation_id"`
	CustomerID     string    `gorm:"size:191;index" json:"customer_id"`
	SenderID       string    `gorm:"size:191;index" json:"sender_id"`
	SenderName     string    `gorm:"size:255" json:"sender_name"`
	Direction      string    `gorm:"size:20;index;not null" json:"direction"`
	Type           string    `gorm:"size:30;index" json:"type"`
	Text           string    `gorm:"type:text" json:"text"`
	HasPhone       bool      `gorm:"not null;default:false" json:"has_phone"`
	InsertedAt     time.Time `gorm:"index;not null" json:"inserted_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PancakeConversion connects a chat with a sale from this backend, Pancake POS,
// or another order system. ExternalOrderID makes repeated callbacks idempotent.
type PancakeConversion struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ExternalOrderID string    `gorm:"size:191;uniqueIndex;not null" json:"external_order_id"`
	PageID          string    `gorm:"size:191;index;not null" json:"page_id"`
	ConversationID  string    `gorm:"size:191;index;not null" json:"conversation_id"`
	CustomerID      string    `gorm:"size:191;index" json:"customer_id,omitempty"`
	CampaignID      string    `gorm:"size:191;index" json:"campaign_id,omitempty"`
	CampaignName    string    `gorm:"size:255" json:"campaign_name,omitempty"`
	ProductName     string    `gorm:"size:255;index" json:"product_name,omitempty"`
	Amount          int64     `gorm:"not null;default:0" json:"amount"`
	ConvertedAt     time.Time `gorm:"index;not null" json:"converted_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
