package seeders

import (
	"fmt"

	"begmt2/models"

	"gorm.io/gorm"
)

type StaffAccount struct {
	Name     string
	Email    string
	Password string
	Role     models.Role
}

func SeedStaffAccounts(db *gorm.DB, accounts []StaffAccount) error {
	for _, account := range accounts {
		if account.Email == "" || account.Password == "" {
			return fmt.Errorf("staff account %s has empty email or password", account.Name)
		}
		if !models.IsValidRole(account.Role) {
			return fmt.Errorf("staff account %s has invalid role %s", account.Email, account.Role)
		}

		data := defaultUser{
			Name:     account.Name,
			Email:    account.Email,
			Password: account.Password,
			Role:     account.Role,
		}
		if err := seedDefaultUser(db, data); err != nil {
			return fmt.Errorf("failed to seed staff account %s: %w", account.Email, err)
		}
	}

	return nil
}
