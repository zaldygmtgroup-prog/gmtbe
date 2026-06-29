package services

import (
	"crypto/tls"
	"fmt"
	"log"

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
	// If credentials are empty or contain placeholder values, fallback to logging in development
	isPlaceholder := s.cfg.MailUsername == "" ||
		s.cfg.MailUsername == "your-gmail@gmail.com" ||
		s.cfg.MailPassword == "" ||
		s.cfg.MailPassword == "your-gmail-app-password"

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
	if s.cfg.MailInsecureSkipVerify {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return dialer.DialAndSend(message)
}
