package services

import (
	"fmt"

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
	if s.cfg.MailUsername == "" || s.cfg.MailPassword == "" {
		return fmt.Errorf("mail credentials are not configured")
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
	return dialer.DialAndSend(message)
}
