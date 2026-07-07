package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"begmt2/config"

	"gopkg.in/gomail.v2"
)

type MailService struct {
	cfg config.Config
}

func NewMailService(cfg config.Config) MailService {
	return MailService{cfg: cfg}
}

func (s MailService) SendPasswordResetToken(toEmail, toName, token string) error {
	var isPlaceholder bool
	if s.cfg.MailMailer == "sendgrid" {
		isPlaceholder = s.cfg.SendGridAPIKey == ""
	} else if s.cfg.MailMailer == "resend" {
		isPlaceholder = s.cfg.ResendAPIKey == ""
	} else {
		isPlaceholder = s.cfg.MailUsername == "" ||
			s.cfg.MailUsername == "your-gmail@gmail.com" ||
			s.cfg.MailPassword == "" ||
			s.cfg.MailPassword == "your-gmail-app-password"
	}

	if isPlaceholder {
		if s.cfg.AppEnv == "development" {
			log.Printf("[DEV-MAIL] Password reset email simulation:")
			log.Printf("[DEV-MAIL] To: %s <%s>", toName, toEmail)
			log.Printf("[DEV-MAIL] Subject: Token Reset Password")
			log.Printf("[DEV-MAIL] Token: %s (Expires in %d minutes)", token, s.cfg.ResetTokenExpiresMinutes)
			return nil
		}
		return fmt.Errorf("mail credentials are not configured (placeholder or empty credentials)")
	}

	if s.cfg.MailMailer == "sendgrid" {
		if err := s.sendWithSendGrid(toEmail, toName, token); err != nil {
			return s.fallbackToSMTP(toEmail, toName, token, err)
		}
		return nil
	} else if s.cfg.MailMailer == "resend" {
		if err := s.sendWithResend(toEmail, toName, token); err != nil {
			return s.fallbackToSMTP(toEmail, toName, token, err)
		}
		return nil
	}

	return s.sendWithSMTP(toEmail, toName, token)
}

func (s MailService) sendWithSMTP(toEmail, toName, token string) error {
	message := gomail.NewMessage()
	message.SetHeader("From", message.FormatAddress(s.cfg.MailUsername, s.cfg.MailFromName))
	message.SetHeader("To", message.FormatAddress(toEmail, toName))
	message.SetHeader("Subject", "Token Reset Password")
	message.SetBody("text/html", fmt.Sprintf(`
		<p>Halo %s,</p>
		<p>Gunakan token berikut untuk mengganti password:</p>
		<h2>%s</h2>
		<p>Token berlaku selama %d menit.</p>
		<p>Abaikan email ini jika kamu tidak meminta reset password.</p>
	`, toName, token, s.cfg.ResetTokenExpiresMinutes))

	dialer := gomail.NewDialer(s.cfg.MailHost, s.cfg.MailPort, s.cfg.MailUsername, s.cfg.MailPassword)
	if s.cfg.MailScheme == "smtps" || s.cfg.MailPort == 465 {
		dialer.SSL = true
	} else if s.cfg.MailScheme == "smtp" {
		dialer.SSL = false
	}
	if s.cfg.MailInsecureSkipVerify {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return dialer.DialAndSend(message)
}

func (s MailService) fallbackToSMTP(toEmail, toName, token string, primaryErr error) error {
	if !s.hasSMTPConfig() {
		return primaryErr
	}
	if err := s.sendWithSMTP(toEmail, toName, token); err != nil {
		return fmt.Errorf("%v; smtp fallback failed: %w", primaryErr, err)
	}
	return nil
}

func (s MailService) hasSMTPConfig() bool {
	return s.cfg.MailUsername != "" &&
		s.cfg.MailUsername != "your-gmail@gmail.com" &&
		s.cfg.MailPassword != "" &&
		s.cfg.MailPassword != "your-gmail-app-password"
}

func (s MailService) sendWithSendGrid(toEmail, toName, token string) error {
	url := s.cfg.SendGridAPIURL
	if url == "" {
		url = "https://api.sendgrid.com/v3/mail/send"
	}

	fromEmail := s.cfg.SendGridFromEmail
	if fromEmail == "" {
		fromEmail = s.cfg.MailUsername
	}

	bodyContent := fmt.Sprintf(`
		<p>Halo %s,</p>
		<p>Gunakan token berikut untuk mengganti password:</p>
		<h2>%s</h2>
		<p>Token berlaku selama %d menit.</p>
		<p>Abaikan email ini jika kamu tidak meminta reset password.</p>
	`, toName, token, s.cfg.ResetTokenExpiresMinutes)

	type SendGridEmail struct {
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}

	type SendGridPersonalization struct {
		To      []SendGridEmail `json:"to"`
		Subject string          `json:"subject"`
	}

	type SendGridContent struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}

	type SendGridRequestBody struct {
		Personalizations []SendGridPersonalization `json:"personalizations"`
		From             SendGridEmail             `json:"from"`
		Content          []SendGridContent         `json:"content"`
	}

	reqBody := SendGridRequestBody{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmail{
					{
						Email: toEmail,
						Name:  toName,
					},
				},
				Subject: "Token Reset Password",
			},
		},
		From: SendGridEmail{
			Email: fromEmail,
			Name:  s.cfg.MailFromName,
		},
		Content: []SendGridContent{
			{
				Type:  "text/html",
				Value: bodyContent,
			},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal SendGrid request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create SendGrid HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.SendGridAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request to SendGrid: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendgrid API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s MailService) sendWithResend(toEmail, toName, token string) error {
	url := s.cfg.ResendAPIURL
	if url == "" {
		url = "https://api.resend.com/emails"
	}

	fromEmail := s.cfg.ResendFromEmail
	if fromEmail == "" {
		fromEmail = "onboarding@resend.dev"
	}

	bodyContent := fmt.Sprintf(`
		<p>Halo %s,</p>
		<p>Gunakan token berikut untuk mengganti password:</p>
		<h2>%s</h2>
		<p>Token berlaku selama %d menit.</p>
		<p>Abaikan email ini jika kamu tidak meminta reset password.</p>
	`, toName, token, s.cfg.ResetTokenExpiresMinutes)

	type ResendRequestBody struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
	}

	reqBody := ResendRequestBody{
		From:    fmt.Sprintf("%s <%s>", s.cfg.MailFromName, fromEmail),
		To:      []string{toEmail},
		Subject: "Token Reset Password",
		HTML:    bodyContent,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal Resend request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create Resend HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request to Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
