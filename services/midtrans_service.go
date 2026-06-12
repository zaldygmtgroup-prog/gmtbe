package services

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"begmt2/config"
	"begmt2/models"
)

type MidtransService struct {
	cfg        config.Config
	httpClient *http.Client
}

type MidtransItem struct {
	ID       string `json:"id"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type MidtransCustomer struct {
	FirstName string `json:"first_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type MidtransSnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

type MidtransNotification struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
}

func NewMidtransService(cfg config.Config) MidtransService {
	return MidtransService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (m MidtransService) IsEnabled() bool {
	return strings.TrimSpace(m.cfg.MidtransServerKey) != ""
}

func (m MidtransService) CreateSnapTransaction(orderID string, grossAmount int64, customer MidtransCustomer, items []MidtransItem) (MidtransSnapResponse, error) {
	if !m.IsEnabled() {
		return MidtransSnapResponse{}, errors.New("midtrans server key is not configured")
	}

	payload := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     orderID,
			"gross_amount": grossAmount,
		},
		"customer_details": customer,
		"item_details":     items,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return MidtransSnapResponse{}, err
	}

	req, err := http.NewRequest(http.MethodPost, m.snapURL(), bytes.NewReader(body))
	if err != nil {
		return MidtransSnapResponse{}, err
	}
	req.SetBasicAuth(m.cfg.MidtransServerKey, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return MidtransSnapResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return MidtransSnapResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return MidtransSnapResponse{}, fmt.Errorf("midtrans snap request failed: status %d body %s", resp.StatusCode, string(respBody))
	}

	var snapResp MidtransSnapResponse
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return MidtransSnapResponse{}, err
	}
	if snapResp.RedirectURL == "" || snapResp.Token == "" {
		return MidtransSnapResponse{}, errors.New("midtrans snap response is missing token or redirect url")
	}

	return snapResp, nil
}

func (m MidtransService) VerifyNotificationSignature(n MidtransNotification) bool {
	if !m.IsEnabled() {
		return false
	}
	raw := n.OrderID + n.StatusCode + n.GrossAmount + m.cfg.MidtransServerKey
	hash := sha512.Sum512([]byte(raw))
	return strings.EqualFold(hex.EncodeToString(hash[:]), n.SignatureKey)
}

func (m MidtransService) MapPaymentStatus(n MidtransNotification) models.PaymentStatus {
	switch n.TransactionStatus {
	case "capture":
		if n.FraudStatus == "challenge" {
			return models.PaymentStatusPending
		}
		return models.PaymentStatusPaid
	case "settlement":
		return models.PaymentStatusPaid
	case "pending":
		return models.PaymentStatusPending
	case "expire":
		return models.PaymentStatusExpired
	case "cancel", "deny", "failure":
		return models.PaymentStatusFailed
	case "refund", "partial_refund":
		return models.PaymentStatusRefund
	default:
		return models.PaymentStatusPending
	}
}

func (m MidtransService) snapURL() string {
	if strings.EqualFold(m.cfg.MidtransEnvironment, "production") {
		return "https://app.midtrans.com/snap/v1/transactions"
	}
	return "https://app.sandbox.midtrans.com/snap/v1/transactions"
}
