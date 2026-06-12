package controllers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"begmt2/models"
)

func TestGeneratePreorderPDFPreview(t *testing.T) {
	if os.Getenv("PDF_PREVIEW") != "1" {
		t.Skip("set PDF_PREVIEW=1 to generate the preorder PDF preview")
	}

	preorder := models.Preorder{
		ID:            1,
		PONumber:      "PO-PREVIEW-001",
		NamaCustomer:  "PT Contoh Customer",
		Email:         "purchasing@example.com",
		NoHP:          "+62 812-3456-7890",
		Alamat:        "Jl. Contoh Raya No. 10, Jakarta",
		Catatan:       "Sample preview untuk validasi layout quotation PDF.",
		Subtotal:      954320000,
		TotalDiscount: 193120000,
		Total:         761200000,
		TotalKomisi:   38060000,
		Status:        models.PreorderStatusInReview,
		PaymentStatus: models.PaymentStatusPending,
		PaymentURL:    "https://app.sandbox.midtrans.com/snap/v4/redirection/064f40f8-211f-44e0-8d51-390cdbec7b90",
		CreatedAt:     time.Date(2026, 6, 12, 10, 30, 0, 0, time.Local),
		Agent: &models.User{
			Name: "Official Agent Preview",
		},
		Items: []models.PreorderItem{
			{
				ProductNameSnapshot:        "MOXLITE ARES",
				ProductDescriptionSnapshot: "*380W PHILIPS MSD Silver 380/2 LL lamp source with 7800K color temperature, CRI 80, and 4000-hour lifespan.\n*Pan 540° and tilt 270° with quiet and precise 3-phase motors.\n*Magnetic coding for accurate positioning.\n*Input voltage AC100-240V 50/60Hz.\n*Frost filter for hybrid wash effect.",
				UnitSnapshot:               "Unit",
				UnitPrice:                  16240000,
				Qty:                        30,
				DiscountPercent:            20,
				DiscountAmount:             97440000,
				Subtotal:                   487200000,
				Total:                      389760000,
				Komisi:                     19488000,
			},
			{
				ProductNameSnapshot:        "MOXLITE SCARLET HYBRID",
				ProductDescriptionSnapshot: "*600W LED module engine with 8500K color temperature.\n*Pan 540° and tilt 270° with quiet and precise motors.\n*Power supply 200W consumption.\n*Linear CMY color mixing system with fixed color wheel.\n*Electronic focus and prism effects.",
				UnitSnapshot:               "Unit",
				UnitPrice:                  32110000,
				Qty:                        20,
				DiscountPercent:            20,
				DiscountAmount:             128440000,
				Subtotal:                   642200000,
				Total:                      513760000,
				Komisi:                     25688000,
			},
		},
	}

	outDir := filepath.Join("..", "test_output")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	outPath := filepath.Join(outDir, "preorder_preview.pdf")
	pdf := buildPreorderPDF(preorder)
	if err := pdf.OutputFileAndClose(outPath); err != nil {
		t.Fatalf("failed to write preview PDF: %v", err)
	}
	t.Logf("generated %s", outPath)
}
