package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func (s *PancakeService) SendPaymentInstructions(po models.Preorder) error {
	if s.cfg.PancakePageID == "" || s.cfg.PancakePageAccessToken == "" {
		return fmt.Errorf("pancake configuration is not set")
	}

	phone := po.NoHP
	if phone == "" {
		return fmt.Errorf("customer phone number is empty")
	}

	// Convert local prefix 0 to country code 62 (assuming Indonesian numbers) if necessary,
	// though standard WABA usually requires 62.
	if len(phone) > 0 && phone[0] == '0' {
		phone = "62" + phone[1:]
	}

	// Follow yaml tip: [pageID]_[phoneNumber]
	conversationID := fmt.Sprintf("%s_%s", s.cfg.PancakePageID, phone)

	url := fmt.Sprintf("https://pages.fm/api/public_api/v1/pages/%s/conversations/%s/messages?page_access_token=%s",
		s.cfg.PancakePageID, conversationID, s.cfg.PancakePageAccessToken)

	var payload map[string]interface{}

	if s.cfg.PancakeWATemplateID != "" {
		// Send WhatsApp Template Message
		payload = map[string]interface{}{
			"action":      "reply_inbox",
			"template_id": s.cfg.PancakeWATemplateID,
			// Simplified template params, to be adjusted based on the actual template in WA Business
			"template_params": map[string]interface{}{
				"BODY_PARAMS": map[string]interface{}{
					"customer_name": po.NamaCustomer,
					"po_number":     po.PONumber,
				},
			},
		}
	} else {
		// Send normal Inbox Message if template is not configured
		message := fmt.Sprintf("Halo %s,\n\nPreorder Anda dengan nomor %s telah disetujui. Total yang harus dibayar: Rp %d.\nSilakan balas pesan ini untuk mendapatkan informasi rekening dan cara pembayaran.\n\nTerima kasih.", po.NamaCustomer, po.PONumber, po.Total)
		
		payload = map[string]interface{}{
			"action":  "reply_inbox",
			"message": message,
		}
	}

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

	return nil
}
