package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONField is a generic helper to marshal/unmarshal a struct/slice to/from JSON in database columns.
type JSONField[T any] struct {
	Val T
}

func (jf *JSONField[T]) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan JSONField: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, &jf.Val)
}

func (jf JSONField[T]) Value() (driver.Value, error) {
	bytes, err := json.Marshal(jf.Val)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (jf JSONField[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(jf.Val)
}

func (jf *JSONField[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &jf.Val)
}

type ContentBriefItem struct {
	Day         string                 `json:"day"`
	Format      string                 `json:"format"`
	Pillar      string                 `json:"pillar"`
	Objective   string                 `json:"objective"`
	Idea        string                 `json:"idea"`
	FormatGuide string                 `json:"formatGuide"`
	Action      string                 `json:"action"`
	Reason      string                 `json:"reason"`
	Impact      string                 `json:"impact"`
	Assistant   map[string]interface{} `json:"assistant,omitempty"`
}

type ContentBrief struct {
	Source  string             `json:"source"`
	Summary string             `json:"summary"`
	Items   []ContentBriefItem `json:"items"`
}

type ContentReasoning struct {
	MediaID   string `json:"media_id"`
	Reasoning string `json:"reasoning"`
	Action    string `json:"action"`
	Angle     string `json:"angle"`
}

type ContentReference struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ContentType string `json:"contentType"`
	Hook        string `json:"hook"`
	Style       string `json:"style,omitempty"`
	Reasoning   string `json:"reasoning"`
	Action      string `json:"action"`
	Pillar      string `json:"pillar"`
	Source      string `json:"source"`
}

type AIContentBriefCache struct {
	ID                uint                          `gorm:"primaryKey" json:"id"`
	IGUserID          string                        `gorm:"size:50;not null;uniqueIndex:idx_ai_content_brief_cache_ig_user_id" json:"ig_user_id"`
	IGUsername        string                        `gorm:"size:100" json:"ig_username"`
	ContentBrief      JSONField[ContentBrief]       `gorm:"type:json" json:"content_brief"`
	ContentReasoning  JSONField[[]ContentReasoning] `gorm:"type:json" json:"content_reasoning"`
	ContentReferences JSONField[[]ContentReference] `gorm:"type:json" json:"content_references"`
	GeneratedAt       time.Time                     `gorm:"not null" json:"generated_at"`
	ExpiresAt         time.Time                     `gorm:"not null;index:idx_ai_content_brief_cache_expires_at" json:"expires_at"`
	CreatedAt         time.Time                     `json:"created_at"`
	UpdatedAt         time.Time                     `json:"updated_at"`
}

func (AIContentBriefCache) TableName() string {
	return "ai_content_brief_cache"
}
