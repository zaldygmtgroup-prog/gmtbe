package main

import (
	"log"
	"os"

	"begmt2/config"
	"begmt2/models"
	"begmt2/seeders"
)

func main() {
	cfg := config.Load()
	db := config.ConnectDatabase(cfg)

	if err := db.AutoMigrate(
		&models.User{},
		&models.DetailUser{},
	); err != nil {
		log.Fatalf("failed to migrate staff account tables: %v", err)
	}

	accounts := []seeders.StaffAccount{
		{
			Name:     getEnv("NEW_ADMIN_NAME", "Admin Baru"),
			Email:    getEnv("NEW_ADMIN_EMAIL", "adminbaru@example.com"),
			Password: getEnv("NEW_ADMIN_PASSWORD", "AdminBaru123!"),
			Role:     models.RoleSuperAdmin,
		},
		{
			Name:     getEnv("NEW_SALES_NAME", "Sales Baru"),
			Email:    getEnv("NEW_SALES_EMAIL", "salesbaru@example.com"),
			Password: getEnv("NEW_SALES_PASSWORD", "SalesBaru123!"),
			Role:     models.RoleSales,
		},
	}

	if err := seeders.SeedStaffAccounts(db, accounts); err != nil {
		log.Fatalf("failed to seed staff accounts: %v", err)
	}

	for _, account := range accounts {
		log.Printf("seeded staff account: email=%s role=%s", account.Email, account.Role)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
