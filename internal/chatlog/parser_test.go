package chatlog

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")

	sample := `12/03/2024 08.15 - Pesan dan panggilan dienkripsi secara end-to-end.
12/03/2024 08.16 - Dia: Pagi! Gimana kabar hari ini?
12/03/2024 08.17 - Kamu: Pagi juga. Aman kok.
12/03/2024 08.18 - Dia: Bagus deeeh kalau gitu wkwk.
Nanti siang mau makan apa?
Jangan lupa makan yaaa 😊
12/03/2024 08.20 - Dia: <Media tidak disertakan>
12/03/2024 08.21 - Dia: Ini fotonya lucuu bgt kan
13/03/2024, 09:30 - Dia: Haii udah bangun belom?
13/03/24, 9:35 AM - Dia: Test 12 jam format
14/03/24 14.05 - Dia: Siang!
14/03/24 14.10 - Dia: Kamu keluar dari grup
`

	tests := []struct {
		name          string
		senderFilter  string
		expectedCount int
	}{
		{
			name:          "Ambil semua pesan pengguna",
			senderFilter:  "",
			expectedCount: 7, // 1 Dia, 1 Kamu, 1 Dia multi-line, 1 Dia, 1 Dia, 1 Dia (12h), 1 Dia (Siang) -> total 7 valid messages
		},
		{
			name:          "Filter khusus pengirim Dia",
			senderFilter:  "Dia",
			expectedCount: 6, // Semua milik Dia kecuali Kamu, sistem, media
		},
		{
			name:          "Filter khusus pengirim Kamu",
			senderFilter:  "Kamu",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, err := Parse(strings.NewReader(sample), Options{
				SenderFilter: tt.senderFilter,
				Location:     loc,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(msgs) != tt.expectedCount {
				t.Errorf("got %d messages, want %d", len(msgs), tt.expectedCount)
				for i, m := range msgs {
					t.Logf("[%d] %s (%s): %s", i, m.Time.Format(time.RFC3339), m.Sender, m.Text)
				}
			}
		})
	}
}

func TestParseSampleFile(t *testing.T) {
	f, err := os.Open("../../testdata/sample_chat.txt")
	if err != nil {
		t.Fatalf("failed to open sample file: %v", err)
	}
	defer f.Close()

	msgs, err := Parse(f, Options{
		SenderFilter: "Dia",
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(msgs) == 0 {
		t.Fatal("expected messages from Dia, got 0")
	}

	for _, m := range msgs {
		if m.Sender != "Dia" {
			t.Errorf("unexpected sender: %s", m.Sender)
		}
		if strings.Contains(strings.ToLower(m.Text), "media tidak disertakan") {
			t.Errorf("media placeholder should be excluded: %s", m.Text)
		}
	}
}
