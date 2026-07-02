package services

import "testing"

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
