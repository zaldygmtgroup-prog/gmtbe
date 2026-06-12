package models

import "time"

type Product struct {
	IDProduct   uint      `gorm:"primaryKey;column:id_product" json:"id"`
	NameProduct string    `gorm:"size:150;not null;column:namaproduct" json:"namaproduct"`
	Photo       string    `gorm:"size:255;column:foto" json:"foto"`
	Description string    `gorm:"type:text;column:deskripsi" json:"deskripsi"`
	Unit        string    `gorm:"size:50;not null" json:"unit"`
	Price       int64     `gorm:"not null" json:"price"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
