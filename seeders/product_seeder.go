package seeders

import (
	"log"

	"begmt2/models"

	"gorm.io/gorm"
)

type rateCardProduct struct {
	Name   string
	Price  int64
	Komisi float64
}

func SeedProducts(db *gorm.DB) {
	products := []rateCardProduct{
		{Name: "Moxlite Amos", Price: 8800000, Komisi: 462000},
		{Name: "Moxlite Amos Plus", Price: 11090000, Komisi: 582000},
		{Name: "Moxlite Amos Pro", Price: 16570000, Komisi: 869000},
		{Name: "Moxlite Ares", Price: 17900000, Komisi: 939000},
		{Name: "Moxlite Scarlet", Price: 12130000, Komisi: 636000},
		{Name: "Moxlite Scarlet Plus", Price: 19250000, Komisi: 1010000},
		{Name: "Moxlite Scarlet Hybrid", Price: 35400000, Komisi: 1858000},
		{Name: "Moxlite IP Scarlet Hybrid", Price: 48310000, Komisi: 2536000},
		{Name: "Moxlite Hera Lite", Price: 7600000, Komisi: 399000},
		{Name: "Moxlite Medusa Lite", Price: 7950000, Komisi: 417000},
		{Name: "Moxlite IP Medusa Lite", Price: 13080000, Komisi: 686000},
		{Name: "Moxlite Medusa Plus", Price: 15750000, Komisi: 826000},
		{Name: "Moxlite IP Medusa Plus", Price: 19140000, Komisi: 1004000},
		{Name: "Moxlite IP Medusa Pro", Price: 17360000, Komisi: 911000},
		{Name: "Moxlite Catrice", Price: 7200000, Komisi: 378000},
		{Name: "Moxlite Catrice Hybrid", Price: 12240000, Komisi: 642000},
		{Name: "Moxlite Catrice Hybrid Pro", Price: 13780000, Komisi: 723000},
		{Name: "Moxlite IP Catrice Hybrid Pro", Price: 16970000, Komisi: 890000},
		{Name: "Moxlite Jim Base", Price: 9030000, Komisi: 474000},
		{Name: "Moxlite Jim", Price: 17700000, Komisi: 929000},
		{Name: "Moxlite Jim Hybrid", Price: 19450000, Komisi: 1021000},
		{Name: "Moxlite Maxine", Price: 9850000, Komisi: 517000},
		{Name: "Moxlite Hexa Eye", Price: 14900000, Komisi: 782000},
		{Name: "Moxlite Hexa Line", Price: 13190000, Komisi: 692000},
		{Name: "Moxlite Holystorm Lite", Price: 5360000, Komisi: 281000},
		{Name: "Moxlite Holystorm", Price: 6030000, Komisi: 316000},
		{Name: "Moxlite IP Holystorm Hybrid", Price: 10540000, Komisi: 553000},
		{Name: "Moxlite IP Holystorm", Price: 16400000, Komisi: 861000},
		{Name: "Moxlite IP Holystorm Bar", Price: 7530000, Komisi: 395000},
		{Name: "Moxlite Berlin", Price: 5550000, Komisi: 291000},
		{Name: "Moxlite Berlin Hybrid", Price: 4760000, Komisi: 249000},
		{Name: "Moxlite IP Berlin", Price: 7160000, Komisi: 375000},
		{Name: "Moxlite Optic", Price: 3010000, Komisi: 158000},
		{Name: "Moxlite Optic Plus", Price: 5020000, Komisi: 263000},
		{Name: "Moxlite Studio Basic", Price: 9220000, Komisi: 484000},
		{Name: "Moxlite Studio Basic Plus", Price: 10110000, Komisi: 530000},
		{Name: "Moxlite Studio Profile", Price: 17640000, Komisi: 926000},
		{Name: "Moxlite Studio Hybrid", Price: 15120000, Komisi: 793000},
		{Name: "Moxlite Studio Performance", Price: 14840000, Komisi: 779000},
		{Name: "Moxlite Hades VI", Price: 24340000, Komisi: 1277000},
		{Name: "Moxlite Hades X", Price: 26780000, Komisi: 1405000},
		{Name: "Moxlite Hades XX", Price: 68940000, Komisi: 3619000},
		{Name: "Moxlite Hades XXX", Price: 108860000, Komisi: 5715000},
		{Name: "Moxlite Parled 544", Price: 4380000, Komisi: 229000},
		{Name: "Moxlite Parled 715", Price: 3210000, Komisi: 168000},
		{Name: "Moxlite Parled 715 IP", Price: 4400000, Komisi: 231000},
		{Name: "Moxlite Magic Panel", Price: 9000000, Komisi: 472000},
		{Name: "Electric Brain Smart Splitter", Price: 3690000, Komisi: 193000},
		{Name: "Electric Brain Splitter Dual Mode", Price: 2970000, Komisi: 155000},
		{Name: "Electric Brain Artnet 8CH", Price: 7830000, Komisi: 411000},
		{Name: "NPU", Price: 33840000, Komisi: 1776000},
		{Name: "Pangolin FB4", Price: 30680000, Komisi: 1610000},
		{Name: "Moxlite Thunder P60", Price: 10650000, Komisi: 559000},
	}

	for _, p := range products {
		var existing models.Product
		if err := db.Where("namaproduct = ?", p.Name).First(&existing).Error; err == nil {
			// Update if exists
			updates := map[string]interface{}{
				"price":  p.Price,
				"komisi": p.Komisi,
				"status": "active",
			}
			if err := db.Model(&existing).Updates(updates).Error; err != nil {
				log.Printf("failed to update product %s: %v", p.Name, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// Create if not exists
			newProduct := models.Product{
				NameProduct: p.Name,
				Price:       p.Price,
				Komisi:      p.Komisi,
				Unit:        "pcs",
				Status:      "active",
				Description: "Product description for " + p.Name,
				Photo:       "",
			}
			if err := db.Create(&newProduct).Error; err != nil {
				log.Printf("failed to create product %s: %v", p.Name, err)
			}
		} else {
			log.Printf("database error checking product %s: %v", p.Name, err)
		}
	}
}
