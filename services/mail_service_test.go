package services_test

import (
	"begmt2/config"
	"begmt2/services"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestSendPasswordResetToken_SendGrid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v3/mail/send" {
			t.Errorf("expected /v3/mail/send path, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer mock-api-key" {
			t.Errorf("expected Bearer mock-api-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body struct {
			Personalizations []struct {
				To []struct {
					Email string `json:"email"`
					Name  string `json:"name"`
				} `json:"to"`
				Subject string `json:"subject"`
			} `json:"personalizations"`
			From struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"from"`
			Content []struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if len(body.Personalizations) != 1 || len(body.Personalizations[0].To) != 1 {
			t.Fatalf("unexpected personalizations structure")
		}
		to := body.Personalizations[0].To[0]
		if to.Email != "recipient@example.com" || to.Name != "Recipient Name" {
			t.Errorf("unexpected recipient: %+v", to)
		}
		if body.Personalizations[0].Subject != "Token Reset Password" {
			t.Errorf("unexpected subject: %s", body.Personalizations[0].Subject)
		}
		if body.From.Email != "sender@example.com" || body.From.Name != "BeGMT2" {
			t.Errorf("unexpected sender: %+v", body.From)
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	cfg := config.Config{
		AppEnv:                   "production",
		MailMailer:               "sendgrid",
		SendGridAPIKey:           "mock-api-key",
		SendGridFromEmail:        "sender@example.com",
		SendGridAPIURL:           server.URL + "/v3/mail/send",
		MailFromName:             "BeGMT2",
		ResetTokenExpiresMinutes: 15,
	}

	mailService := services.NewMailService(cfg)
	err := mailService.SendPasswordResetToken("recipient@example.com", "Recipient Name", "123456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSendPasswordResetToken_Resend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/emails" {
			t.Errorf("expected /emails path, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer mock-resend-key" {
			t.Errorf("expected Bearer mock-resend-key, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			HTML    string   `json:"html"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if body.From != "BeGMT2 <sender@example.com>" {
			t.Errorf("unexpected sender: %s", body.From)
		}
		if len(body.To) != 1 || body.To[0] != "recipient@example.com" {
			t.Errorf("unexpected recipient: %+v", body.To)
		}
		if body.Subject != "Token Reset Password" {
			t.Errorf("unexpected subject: %s", body.Subject)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.Config{
		AppEnv:                   "production",
		MailMailer:               "resend",
		ResendAPIKey:             "mock-resend-key",
		ResendFromEmail:          "sender@example.com",
		ResendAPIURL:             server.URL + "/emails",
		MailFromName:             "BeGMT2",
		ResetTokenExpiresMinutes: 15,
	}

	mailService := services.NewMailService(cfg)
	err := mailService.SendPasswordResetToken("recipient@example.com", "Recipient Name", "123456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

