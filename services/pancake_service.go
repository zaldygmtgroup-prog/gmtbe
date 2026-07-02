package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"begmt2/config"
	"begmt2/models"
)

type PancakeService struct {
	cfg config.Config
}

func NewPancakeService(cfg config.Config) *PancakeService {
	return &PancakeService{cfg: cfg}
}

func (s *PancakeService) SendTextMessage(phone, message string) error {
	if s.cfg.PancakePageID == "" || s.cfg.PancakePageAccessToken == "" {
		return fmt.Errorf("pancake configuration is not set")
	}

	phone = normalizePancakePhone(phone)
	if phone == "" {
		return fmt.Errorf("phone number is empty")
	}

	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("message is empty")
	}

	return s.sendMessagePayload(phone, map[string]interface{}{
		"action":  "reply_inbox",
		"message": message,
	})
}

func (s *PancakeService) SendTemplateMessage(phone, templateID string, templateParams map[string]interface{}) error {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return fmt.Errorf("template id is empty")
	}

	payload := map[string]interface{}{
		"action":      "reply_inbox",
		"template_id": templateID,
	}
	if len(templateParams) > 0 {
		payload["template_params"] = templateParams
	}

	return s.sendMessagePayload(phone, payload)
}

func (s *PancakeService) SendPasswordResetToken(phone, name, token string, expiresMinutes int) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Pengguna"
	}

	if s.cfg.PancakeResetTemplateID != "" {
		return s.SendTemplateMessage(phone, s.cfg.PancakeResetTemplateID, map[string]interface{}{
			"BODY_PARAMS": map[string]string{
				"name":            name,
				"token":           token,
				"expires_minutes": strconv.Itoa(expiresMinutes),
			},
		})
	}

	message := fmt.Sprintf("Halo %s,\n\nToken reset password BeGMT2 Anda: %s\nToken berlaku selama %d menit. Jangan bagikan token ini kepada siapa pun.\n\nJika Anda tidak meminta reset password, abaikan pesan ini.", name, token, expiresMinutes)
	return s.SendTextMessage(phone, message)
}

func (s *PancakeService) SendPaymentInstructions(po models.Preorder) error {
	phone := normalizePancakePhone(po.NoHP)
	if phone == "" {
		return fmt.Errorf("customer phone number is empty")
	}

	if s.cfg.PancakeWATemplateID != "" {
		return s.sendMessagePayload(phone, map[string]interface{}{
			"action":      "reply_inbox",
			"template_id": s.cfg.PancakeWATemplateID,
		})
	}

	message := fmt.Sprintf("Halo %s,\n\nPreorder Anda dengan nomor %s telah disetujui. Total yang harus dibayar: Rp %d.\nSilakan balas pesan ini untuk mendapatkan informasi rekening dan cara pembayaran.\n\nTerima kasih.", po.NamaCustomer, po.PONumber, po.Total)
	return s.SendTextMessage(phone, message)
}

func (s *PancakeService) sendMessagePayload(phone string, payload map[string]interface{}) error {
	if s.cfg.PancakePageID == "" || s.cfg.PancakePageAccessToken == "" {
		return fmt.Errorf("pancake configuration is not set")
	}

	phone = normalizePancakePhone(phone)
	if phone == "" {
		return fmt.Errorf("phone number is empty")
	}

	conversationID := fmt.Sprintf("%s_%s", s.cfg.PancakePageID, phone)
	url := fmt.Sprintf("https://pages.fm/api/public_api/v1/pages/%s/conversations/%s/messages?page_access_token=%s",
		s.cfg.PancakePageID, conversationID, s.cfg.PancakePageAccessToken)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("pancake API error, status: %d", resp.StatusCode)
	}

	// Parse JSON response to check for "success": false
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if success, ok := result["success"].(bool); ok && !success {
			msg, _ := result["message"].(string)
			return fmt.Errorf("pancake API rejected message: %s", msg)
		}
	}

	return nil
}

func normalizePancakePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")

	var builder strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}

	phone = builder.String()
	if strings.HasPrefix(phone, "0") {
		phone = "62" + strings.TrimPrefix(phone, "0")
	}

	return phone
}
