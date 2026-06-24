package models

import (
	"time"

	"gorm.io/gorm"
)

type EducationRegistration struct {
	ID                     string         `gorm:"primaryKey;column:id" json:"id"`
	EventID                string         `gorm:"column:event_id" json:"event_id"`
	UserID                 uint           `gorm:"column:user_id" json:"user_id"`
	Salutation             string         `gorm:"column:salutation" json:"salutation"` // Ms, Mr, Mx
	FirstName              string         `gorm:"column:first_name" json:"first_name"`
	Surname                string         `gorm:"column:surname" json:"surname"`
	Email                  string         `gorm:"column:email" json:"email"`
	PhoneLandline          string         `gorm:"column:phone_landline" json:"phone_landline"`
	PhoneMobile            string         `gorm:"column:phone_mobile" json:"phone_mobile"`
	Company                string         `gorm:"column:company" json:"company"`
	Position               string         `gorm:"column:position" json:"position"`
	Street                 string         `gorm:"column:street" json:"street"`
	Postcode               string         `gorm:"column:postcode" json:"postcode"`
	Town                   string         `gorm:"column:town" json:"town"`
	Country                string         `gorm:"column:country" json:"country"`
	MealPreference         string         `gorm:"column:meal_preference" json:"meal_preference"` // None, Vegetarian, Vegan
	AdditionalInformation  string         `gorm:"type:text;column:additional_information" json:"additional_information"`
	ConditionsOfParticipation bool        `gorm:"column:conditions_of_participation" json:"conditions_of_participation"`
	PrivacyPolicy          bool           `gorm:"column:privacy_policy" json:"privacy_policy"`
	MarketingUpdates       bool           `gorm:"column:marketing_updates" json:"marketing_updates"`
	Status                 string         `gorm:"column:status" json:"status"` // Confirmed, Pending, Cancelled
	CreatedAt              time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt              time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index;column:deleted_at" json:"-"`

	Event Education `gorm:"foreignKey:EventID;references:ID" json:"event,omitempty"`
	User  User      `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}
