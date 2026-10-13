package shadow

import (
	"testing"
)

func TestParseAIReco(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "BELI with emoji",
			text: "🟢 **REKOMENDASI: BELI**\n_Alasan: ..._",
			want: "BELI",
		},
		{
			name: "TAHAN uppercase",
			text: "REKOMENDASI: TAHAN",
			want: "TAHAN",
		},
		{
			name: "HINDARI lowercase",
			text: "rekomendasi: hindari",
			want: "HINDARI",
		},
		{
			name: "broken text returns NA",
			text: "Tidak ada rekomendasi di sini",
			want: "NA",
		},
		{
			name: "empty text returns NA",
			text: "",
			want: "NA",
		},
		{
			name: "with asterisks",
			text: "🎯 **REKOMENDASI: BELI** _Alasan: bagus_",
			want: "BELI",
		},
		{
			name: "NA keyword returns NA",
			text: "REKOMENDASI: NA",
			want: "NA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAIReco(tt.text)
			if got != tt.want {
				t.Errorf("ParseAIReco(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestParseConfidence(t *testing.T) {
	tests := []struct {
		name string
		text string
		want float64
	}{
		{
			name: "standard format",
			text: "🤖 **AI Confidence:** 85%",
			want: 85,
		},
		{
			name: "with asterisk",
			text: "AI Confidence:* 72%",
			want: 72,
		},
		{
			name: "decimal",
			text: "AI Confidence: 80.5%",
			want: 80.5,
		},
		{
			name: "broken returns 0",
			text: "Tidak ada confidence di sini",
			want: 0,
		},
		{
			name: "empty returns 0",
			text: "",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseConfidence(tt.text)
			if got != tt.want {
				t.Errorf("ParseConfidence(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}
