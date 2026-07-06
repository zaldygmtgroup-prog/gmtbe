package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	pageID := os.Getenv("PANCAKE_PAGE_ID")
	token := os.Getenv("PANCAKE_PAGE_ACCESS_TOKEN")
	phone := "6281319642511"
	templateID := "1523095815953461"

	payload := map[string]interface{}{
		"action":      "reply_inbox",
		"template_id": templateID,
		"template_params": map[string]interface{}{
			"HEADER_PARAMS": map[string]interface{}{
				"DOCUMENT": map[string]string{
					"url":  "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf",
					"name": "invoice_INV_GMT_2026_07_0013.pdf",
				},
			},
			"BODY_PARAMS": map[string]string{
				"1": "xsxjbsxjsxj",
				"2": "INV/GMT/2026/07/0013",
				"3": "Rp 206.820.000 (DP 50%)",
			},
		},
	}

	fmt.Printf("Sending WhatsApp template %s to %s...\n", templateID, phone)
	err := sendPayload(pageID, token, phone, payload)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	} else {
		fmt.Println("Success! Check your WhatsApp.")
	}
}

func sendPayload(pageID, token, phone string, payload map[string]interface{}) error {
	conversationID := fmt.Sprintf("%s_%s", pageID, phone)
	url := fmt.Sprintf("https://pages.fm/api/public_api/v1/pages/%s/conversations/%s/messages?page_access_token=%s",
		pageID, conversationID, token)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{}
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

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d, Response: %s\n", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Pancake API returned error status: %d", resp.StatusCode)
	}

	return nil
}
