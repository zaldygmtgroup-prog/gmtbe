package main

import (
	"log"

	"begmt2/config"
	"begmt2/models"
	"begmt2/routes"
	"begmt2/seeders"
)

func main() {
	cfg := config.Load()
	db := config.ConnectDatabase(cfg)

	if err := db.AutoMigrate(
		&models.User{},
		&models.DetailUser{},
		&models.PasswordResetToken{},
		&models.AgentWallet{},
		&models.AgentCommission{},
		&models.WithdrawRequest{},
		&models.Product{},
		&models.Preorder{},
		&models.PreorderItem{},
		&models.Notification{},
		&models.AgentOnboardingVideo{},
		&models.AgentOnboardingProgress{},
		&models.AuthSession{},
		&models.SSOCode{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	seeders.SeedDefaultUsers(db, cfg)
	seeders.SeedOnboardingVideos(db)
	seeders.SeedProducts(db)

	r := routes.SetupRouter(cfg, db)

	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
