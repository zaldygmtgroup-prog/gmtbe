package models

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	IDProduct       uint             `gorm:"primaryKey;column:id_product"        json:"id"`
	NameProduct     string           `gorm:"size:150;not null;column:namaproduct" json:"namaproduct"`
	Photo           string           `gorm:"size:255;column:foto"                json:"foto"`
	Description     string           `gorm:"type:text;column:deskripsi"          json:"deskripsi"`
	Unit            string           `gorm:"size:50;not null;column:unit"        json:"unit"`
	Price           int64            `gorm:"not null;column:price"               json:"price"`
	Status          string           `gorm:"size:50;column:status"               json:"status"`
	Komisi          float64                    `gorm:"column:komisi"                       json:"komisi"`
	CommissionTiers JSONField[map[string]int64] `gorm:"type:json;column:commission_tiers"  json:"commission_tiers"`
	CreatedAt       time.Time                  `gorm:"column:created_at"                   json:"created_at"`
	UpdatedAt       time.Time                  `gorm:"column:updated_at"                   json:"updated_at"`
	DeletedAt       gorm.DeletedAt             `gorm:"index;column:deleted_at"             json:"-"`
}

func (p Product) CalculateCommission(discountPercent float64) int64 {
	if discountPercent == 0 {
		return int64(p.Komisi)
	}
	baseKomisi := p.Komisi
	if baseKomisi <= 0 {
		baseKomisi = float64(p.Price) * 0.0525
	}
	discountAmount := float64(p.Price) * (discountPercent / 100.0)
	penalty := discountAmount * 0.13
	rawCommission := baseKomisi - penalty
	commission := int64(rawCommission / 1000.0) * 1000
	if commission < 0 {
		return 0
	}
	return commission
}

func (p *Product) PopulateCommissionTiers() {
	if p.CommissionTiers.Val == nil || len(p.CommissionTiers.Val) == 0 {
		p.CommissionTiers = JSONField[map[string]int64]{
			Val: map[string]int64{
				"0%":  p.CalculateCommission(0),
				"5%":  p.CalculateCommission(5),
				"10%": p.CalculateCommission(10),
				"15%": p.CalculateCommission(15),
				"20%": p.CalculateCommission(20),
				"25%": p.CalculateCommission(25),
				"28%": p.CalculateCommission(28),
			},
		}
	}
}
