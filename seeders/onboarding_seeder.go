package seeders

import (
	"log"

	"begmt2/models"

	"gorm.io/gorm"
)

type defaultVideo struct {
	Slug            string
	Title           string
	Description     string
	VideoURL        string
	DurationSeconds int
	SortOrder       int
	IsRequired      bool
}

func SeedOnboardingVideos(db *gorm.DB) {
	videos := []defaultVideo{
		{
			Slug:            "agent-introduction",
			Title:           "Pengenalan Role Agent",
			Description:     "Dasar tugas agent...",
			VideoURL:        "https://example.com/videos/agent-intro.mp4",
			DurationSeconds: 380,
			SortOrder:       1,
			IsRequired:      true,
		},
		{
			Slug:            "product-and-po-flow",
			Title:           "Product and PO Flow",
			Description:     "Alur pemesanan produk...",
			VideoURL:        "https://example.com/videos/product-po-flow.mp4",
			DurationSeconds: 420,
			SortOrder:       2,
			IsRequired:      true,
		},
		{
			Slug:            "commission-calculation",
			Title:           "Commission Calculation",
			Description:     "Cara menghitung komisi...",
			VideoURL:        "https://example.com/videos/commission-calc.mp4",
			DurationSeconds: 300,
			SortOrder:       3,
			IsRequired:      true,
		},
	}

	for _, v := range videos {
		if err := seedOnboardingVideo(db, v); err != nil {
			log.Printf("failed to seed onboarding video %s: %v", v.Slug, err)
		}
	}
}

func seedOnboardingVideo(db *gorm.DB, data defaultVideo) error {
	var existingVideo models.AgentOnboardingVideo
	if err := db.Where("slug = ?", data.Slug).First(&existingVideo).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	video := models.AgentOnboardingVideo{
		Slug:            data.Slug,
		Title:           data.Title,
		Description:     data.Description,
		VideoURL:        data.VideoURL,
		DurationSeconds: data.DurationSeconds,
		SortOrder:       data.SortOrder,
		IsRequired:      data.IsRequired,
	}

	return db.Create(&video).Error
}
