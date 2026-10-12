// Package quickadd menggunakan LLM untuk mengekstrak teks bebas menjadi struktur Reminder atau Event.
package quickadd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mini-assisted/internal/llm"
)

// ParsedItem adalah hasil ekstraksi dari teks masukan pengguna.
type ParsedItem struct {
	Type       string    `json:"type"`        // "reminder" atau "event"
	Title      string    `json:"title"`       // Teks pengingat atau judul acara
	StartAtUTC time.Time `json:"start_at_utc"` // Waktu mulai / jatuh tempo (UTC)
	EndAtUTC   time.Time `json:"end_at_utc"`   // Waktu selesai (jika event)
	RepeatRule string    `json:"repeat_rule"` // "", "daily", "weekly"
	Notes      string    `json:"notes"`
}

// Parser mengonversi bahasa natural menjadi ParsedItem.
type Parser struct {
	llmClient *llm.Client
	timeZone  *time.Location
}

// NewParser membuat instance Parser baru.
func NewParser(llmClient *llm.Client, tz *time.Location) *Parser {
	if tz == nil {
		tz, _ = time.LoadLocation("Asia/Jakarta")
		if tz == nil {
			tz = time.Local
		}
	}
	return &Parser{
		llmClient: llmClient,
		timeZone:  tz,
	}
}

type llmOutput struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	DateTime   string `json:"date_time"` // Format: YYYY-MM-DD HH:MM
	DurationM  int    `json:"duration_minutes"`
	RepeatRule string `json:"repeat_rule"`
	Notes      string `json:"notes"`
}

// Parse mengekstrak perintah bahasa Indonesia menjadi data terstruktur.
func (p *Parser) Parse(ctx context.Context, text string) (*ParsedItem, error) {
	nowLocal := time.Now().In(p.timeZone)

	// Fallback sederhana jika tidak ada LLM client
	if p.llmClient == nil {
		return &ParsedItem{
			Type:       "reminder",
			Title:      strings.TrimSpace(text),
			StartAtUTC: nowLocal.Add(time.Hour).UTC(),
		}, nil
	}

	systemPrompt := fmt.Sprintf(`Kamu adalah parser waktu dan tugas cerdas bahasa Indonesia.
Waktu saat ini (Zona %s): %s (Hari: %s).

Tugasmu mengekstrak teks masukan pengguna menjadi JSON dengan struktur:
{
  "type": "reminder" (untuk tugas/pengingat) atau "event" (untuk jadwal/acara pertemuan),
  "title": "nama tugas atau kegiatan tanpa kata perintah pengingat",
  "date_time": "YYYY-MM-DD HH:MM" (konversi ke waktu yang dimaksud relatif terhadap waktu saat ini),
  "duration_minutes": 60 (durasi dalam menit jika event, 0 jika reminder),
  "repeat_rule": "" atau "daily" atau "weekly",
  "notes": "catatan tambahan jika ada"
}

ATURAN:
1. Kembalikan HANYA format JSON valid tanpa format markdown, tanpa petik pembungkus.
2. Jika jam tidak disebutkan secara spesifik:
   - "pagi" -> 08:00
   - "siang" -> 12:00
   - "sore" -> 16:00
   - "malam" -> 19:30
   - "besok" -> tanggal besok jam 09:00 jika tidak ada jam
   - jika tanpa waktu sama sekali -> 1 jam dari waktu saat ini.`,
		p.timeZone.String(),
		nowLocal.Format("2006-01-02 15:04"),
		nowLocal.Format("Monday"),
	)

	userPrompt := fmt.Sprintf("Ekstrak teks ini: %q", text)

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	respText, err := p.llmClient.GenerateText(ctxTimeout, llm.PromptParams{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.1,
	})
	if err != nil {
		return &ParsedItem{
			Type:       "reminder",
			Title:      strings.TrimSpace(text),
			StartAtUTC: nowLocal.Add(time.Hour).UTC(),
		}, nil
	}

	// Bersihkan markdown ```json jika ada
	cleanJSON := strings.TrimSpace(respText)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	var out llmOutput
	if err := json.Unmarshal([]byte(cleanJSON), &out); err != nil {
		return &ParsedItem{
			Type:       "reminder",
			Title:      strings.TrimSpace(text),
			StartAtUTC: nowLocal.Add(time.Hour).UTC(),
		}, nil
	}

	// Parse waktu hasil LLM sesuai zona waktu lokal
	parsedTime, err := time.ParseInLocation("2006-01-02 15:04", out.DateTime, p.timeZone)
	if err != nil {
		parsedTime = nowLocal.Add(time.Hour)
	}

	startUTC := parsedTime.UTC()
	dur := time.Duration(out.DurationM) * time.Minute
	if dur <= 0 {
		dur = time.Hour
	}
	endUTC := startUTC.Add(dur)

	itemType := strings.ToLower(out.Type)
	if itemType != "event" {
		itemType = "reminder"
	}

	title := strings.TrimSpace(out.Title)
	if title == "" {
		title = strings.TrimSpace(text)
	}

	return &ParsedItem{
		Type:       itemType,
		Title:      title,
		StartAtUTC: startUTC,
		EndAtUTC:   endUTC,
		RepeatRule: out.RepeatRule,
		Notes:      out.Notes,
	}, nil
}
