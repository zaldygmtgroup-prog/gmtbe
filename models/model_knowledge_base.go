package models

import "time"

// ModelKnowledgeBase stores admin-defined knowledge base instructions per AI agent role.
// Each role_key maps to one of the agent roles used during content brief generation:
// growthStrategist, marketingSpecialist, conversionCommunityLead.
type ModelKnowledgeBase struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RoleKey   string    `gorm:"size:100;not null;uniqueIndex:idx_model_kb_role_key" json:"role_key"`
	RoleLabel string    `gorm:"size:150" json:"role_label"`
	Content   string    `gorm:"type:text" json:"content"`
	UpdatedBy uint      `gorm:"default:0" json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ModelKnowledgeBase) TableName() string {
	return "model_knowledge_bases"
}
