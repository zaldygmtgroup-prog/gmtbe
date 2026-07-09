package services

import (
	"strings"
	"testing"

	"begmt2/models"
)

func TestNormalizePancakePhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "local indonesian number", input: "0812-3456-7890", want: "6281234567890"},
		{name: "international number with plus", input: "+62812 3456 7890", want: "6281234567890"},
		{name: "already normalized", input: "6281234567890", want: "6281234567890"},
		{name: "empty when no digits", input: "-", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizePancakePhone(tt.input); got != tt.want {
				t.Fatalf("normalizePancakePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildPaymentInstructionMessageIncludesBankAndSteps(t *testing.T) {
	message := buildPaymentInstructionMessage(models.Preorder{
		NamaCustomer: "Budi",
		PONumber:     "PO-001",
	}, "DP 50%", "Rp 1.000.000")

	expectedParts := []string{
		"BCA 6640755855 CV Santri Putra Abuzed",
		"Tata cara pembayaran:",
		"1. Transfer sesuai nominal tagihan ke rekening BCA di atas.",
		"2. Cantumkan nomor PO pada berita/keterangan transfer jika tersedia.",
		"3. Kirim bukti transfer melalui WhatsApp ini agar pembayaran dapat diverifikasi.",
	}
	for _, part := range expectedParts {
		if !strings.Contains(message, part) {
			t.Fatalf("expected message to contain %q, got %q", part, message)
		}
	}
}
