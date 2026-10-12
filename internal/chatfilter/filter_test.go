package chatfilter

import (
	"testing"
	"time"

	"mini-assisted/internal/chatlog"
)

func TestFilter(t *testing.T) {
	cfg := Config{
		SensitiveKeywords: []string{"transfer", "dokter", "berantem", "rahasia"},
		Blocklist:         []string{"Jakarta", "Budi", "Senayan"},
	}

	tests := []struct {
		name           string
		input          string
		wantDropped    bool
		wantQuestion   bool
		expectedInText string
	}{
		{
			name:           "Pesan santai normal lolos",
			input:          "Pagi! Gimana kabarnya hari ini? Udah makan belum?",
			wantDropped:    false,
			wantQuestion:   false,
			expectedInText: "Pagi! Gimana kabarnya hari ini? Udah makan belum?",
		},
		{
			name:           "Pesan topik sensitif transfer dibuang",
			input:          "Tolong transfer 50rb ya ke rekening",
			wantDropped:    true,
			wantQuestion:   false,
			expectedInText: "",
		},
		{
			name:           "Pesan mengandung nomor telepon disanitasi dan diragukan",
			input:          "Hubungi aku di 081234567890 ya nanti",
			wantDropped:    false,
			wantQuestion:   true,
			expectedInText: "Hubungi aku di [NOMOR_TELEPON] ya nanti",
		},
		{
			name:           "Pesan mengandung email dan link URL disanitasi",
			input:          "Kirim ke test@example.com atau cek https://google.com ya",
			wantDropped:    false,
			wantQuestion:   true,
			expectedInText: "Kirim ke [EMAIL] atau cek [URL] ya",
		},
		{
			name:           "Pesan mengandung entitas blocklist disensor",
			input:          "Kemarin ketemu Budi di Senayan seru banget",
			wantDropped:    false,
			wantQuestion:   true,
			expectedInText: "Kemarin ketemu [REDACTED] di [REDACTED] seru banget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := chatlog.Message{
				Time:   time.Now(),
				Sender: "Dia",
				Text:   tt.input,
			}
			res := Filter(msg, cfg)
			if res.IsDropped != tt.wantDropped {
				t.Errorf("IsDropped = %v, want %v", res.IsDropped, tt.wantDropped)
			}
			if !tt.wantDropped {
				if res.IsQuestionable != tt.wantQuestion {
					t.Errorf("IsQuestionable = %v, want %v", res.IsQuestionable, tt.wantQuestion)
				}
				if res.Cleaned.Text != tt.expectedInText {
					t.Errorf("Cleaned.Text = %q, want %q", res.Cleaned.Text, tt.expectedInText)
				}
			}
		})
	}
}
