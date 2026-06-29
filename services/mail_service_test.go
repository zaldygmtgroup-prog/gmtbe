package services_test

import (
	"begmt2/config"
	"begmt2/services"
	"testing"
)

func TestSendPasswordResetToken_DevelopmentPlaceholder(t *testing.T) {
	cfg := config.Config{
		AppEnv:                   "development",
		MailUsername:             "your-gmail@gmail.com",
		MailPassword:             "your-gmail-app-password",
		ResetTokenExpiresMinutes: 15,
	}

	mailService := services.NewMailService(cfg)
	err := mailService.SendPasswordResetToken("test@example.com", "Test User", "123456")
	if err != nil {
		t.Fatalf("expected no error in development mode with placeholder credentials, got: %v", err)
	}
}

func TestSendPasswordResetToken_ProductionPlaceholder(t *testing.T) {
	cfg := config.Config{
		AppEnv:                   "production",
		MailUsername:             "your-gmail@gmail.com",
		MailPassword:             "your-gmail-app-password",
		ResetTokenExpiresMinutes: 15,
	}

	mailService := services.NewMailService(cfg)
	err := mailService.SendPasswordResetToken("test@example.com", "Test User", "123456")
	if err == nil {
		t.Fatal("expected error in production mode with placeholder credentials, got nil")
	}
}

func TestSendPasswordResetToken_MissingCredentials(t *testing.T) {
	cfg := config.Config{
		AppEnv:                   "production",
		MailUsername:             "",
		MailPassword:             "",
		ResetTokenExpiresMinutes: 15,
	}

	mailService := services.NewMailService(cfg)
	err := mailService.SendPasswordResetToken("test@example.com", "Test User", "123456")
	if err == nil {
		t.Fatal("expected error with missing credentials, got nil")
	}
}
