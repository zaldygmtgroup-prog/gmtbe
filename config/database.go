package config

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConnectDatabase(cfg Config) *gorm.DB {
	if dsn, ok := MySQLDSNFromURL(cfg.DatabaseURL); ok {
		return openMySQL(dsn)
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	return openMySQL(dsn)
}

func openMySQL(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	return db
}
