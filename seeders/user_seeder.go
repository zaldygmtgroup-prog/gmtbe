package seeders

import (
	"log"

	"begmt2/config"
	"begmt2/models"
	"begmt2/utils"

	"gorm.io/gorm"
)

type defaultUser struct {
	Name     string
	Email    string
	Password string
	Role     models.Role
}

func SeedDefaultUsers(db *gorm.DB, cfg config.Config) {
	users := []defaultUser{
		{
			Name:     "Super Admin",
			Email:    cfg.DefaultAdminEmail,
			Password: cfg.DefaultAdminPassword,
			Role:     models.RoleSuperAdmin,
		},
		{
			Name:     "Sales",
			Email:    cfg.DefaultSalesEmail,
			Password: cfg.DefaultSalesPassword,
			Role:     models.RoleSales,
		},
	}

	for _, user := range users {
		if err := seedDefaultUser(db, user); err != nil {
			log.Printf("failed to seed default user %s: %v", user.Email, err)
		}
	}
}

func seedDefaultUser(db *gorm.DB, data defaultUser) error {
	hashedPassword, err := utils.HashPassword(data.Password)
	if err != nil {
		return err
	}

	var existingUser models.User
	if err := db.Where("email = ?", data.Email).First(&existingUser).Error; err == nil {
		updates := map[string]interface{}{
			"name":     data.Name,
			"password": hashedPassword,
			"role":     data.Role,
		}

		return db.Model(&existingUser).Updates(updates).Error
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	user := models.User{
		Name:        data.Name,
		TTL:         "-",
		PhoneNumber: "-",
		Gender:      "-",
		Email:       data.Email,
		Domicile:    "-",
		Password:    hashedPassword,
		Role:        data.Role,
		DetailUser: models.DetailUser{
			CompanyName: "-",
		},
	}

	return db.Create(&user).Error
}
