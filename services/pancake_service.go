package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

func (s *PancakeService) SendPaymentInstructions(po models.Preorder, stage models.PaymentStage, amount int64) error {
	phone := normalizePancakePhone(po.NoHP)
	if phone == "" {
		return fmt.Errorf("customer phone number is empty")
	}
	stageLabel := paymentStageLabel(stage)
	if amount <= 0 {
		amount = po.Total
	}

	if s.cfg.PancakeWATemplateID != "" {
		return s.SendTemplateMessage(phone, s.cfg.PancakeWATemplateID, map[string]interface{}{
			"BODY_PARAMS": map[string]string{
				"1": po.NamaCustomer,
				"2": po.PONumber,
				"3": fmt.Sprintf("Rp %d", amount),
				"4": stageLabel,
			},
		})
	}

	message := fmt.Sprintf("Halo %s,\n\nPreorder Anda dengan nomor %s telah disetujui. Tagihan %s yang harus dibayar: Rp %d.\nSilakan balas pesan ini untuk mendapatkan informasi rekening dan cara pembayaran.\n\nTerima kasih.", po.NamaCustomer, po.PONumber, stageLabel, amount)
	return s.SendTextMessage(phone, message)
}

func paymentStageLabel(stage models.PaymentStage) string {
	switch stage {
	case models.PaymentStageDP:
		return "DP 50%"
	case models.PaymentStageRemaining:
		return "pelunasan 50%"
	default:
		return "100%"
	}
}

func (s *PancakeService) SendDocumentMessage(phone, contentID, filename string) error {
	contentID = strings.TrimSpace(contentID)
	if contentID == "" {
		return fmt.Errorf("content id is empty")
	}

	payload := map[string]interface{}{
		"action":      "reply_inbox",
		"content_ids": []string{contentID},
	}
	if strings.TrimSpace(filename) != "" {
		payload["filename"] = filename
	}

	return s.sendMessagePayload(phone, payload)
}

func (s *PancakeService) SendPreorderInvoice(po models.Preorder, pdfBytes []byte, filename string) error {
	phone := normalizePancakePhone(po.NoHP)
	if phone == "" {
		return fmt.Errorf("customer phone number is empty")
	}
	if len(pdfBytes) == 0 {
		return fmt.Errorf("invoice PDF is empty")
	}
	if strings.TrimSpace(filename) == "" {
		filename = "invoice.pdf"
	}

	contentID, err := s.UploadContent(filename, "application/pdf", pdfBytes)
	if err != nil {
		return err
	}

	return s.SendDocumentMessage(phone, contentID, filename)
}

func (s *PancakeService) UploadContent(filename, contentType string, content []byte) (string, error) {
	if s.cfg.PancakePageID == "" || s.cfg.PancakePageAccessToken == "" {
		return "", fmt.Errorf("pancake configuration is not set")
	}
	if len(content) == 0 {
		return "", fmt.Errorf("content is empty")
	}
	if strings.TrimSpace(filename) == "" {
		filename = "document"
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeMultipartFilename(filename))},
		"Content-Type":        {contentType},
	})
	if err != nil {
		return "", err
	}
	if _, err := part.Write(content); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://pages.fm/api/public_api/v1/pages/%s/upload_contents?page_access_token=%s",
		s.cfg.PancakePageID, s.cfg.PancakePageAccessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("pancake upload API error, status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID      string `json:"id"`
		Success *bool  `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Success != nil && !*result.Success {
		return "", fmt.Errorf("pancake upload API rejected content: %s", result.Message)
	}
	if strings.TrimSpace(result.ID) == "" {
		return "", fmt.Errorf("pancake upload API did not return content id")
	}

	return result.ID, nil
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pancake API error, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
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

func escapeMultipartFilename(filename string) string {
	replacer := strings.NewReplacer("\\", "\\\\", `"`, "\\\"")
	return replacer.Replace(filename)
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
