// Package persona memuat style guide persona dan menyediakan peracikan prompt untuk pengingat dan sapaan.
package persona

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"mini-assisted/internal/llm"
)

// Composer bertanggung jawab membuat pesan berbasis persona dengan LLM (dan fallback template).
type Composer struct {
	llmClient   *llm.Client
	personaPath string
	timeZone    *time.Location
}

// NewComposer membuat instance Composer baru.
func NewComposer(llmClient *llm.Client, personaPath string, tz *time.Location) *Composer {
	if tz == nil {
		tz, _ = time.LoadLocation("Asia/Jakarta")
		if tz == nil {
			tz = time.Local
		}
	}
	if personaPath == "" {
		personaPath = "persona/dara.md"
	}
	return &Composer{
		llmClient:   llmClient,
		personaPath: personaPath,
		timeZone:    tz,
	}
}

// loadPersonaContent membaca file style guide persona dari disk.
func (c *Composer) loadPersonaContent() string {
	for _, p := range []string{c.personaPath, "../" + c.personaPath, "../../" + c.personaPath} {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return "Kamu adalah Dara, pasangan yang akrab, perhatian, santai, memanggil 'sayangg' atau 'bep', huruf kecil semua tanpa tanda titik formal."
}

// getTimeGreeting mengembalikan sapaan waktu sesuai aturan zona waktu.
func getTimeGreeting(now time.Time) string {
	h := now.Hour()
	switch {
	case h >= 5 && h < 11:
		return "pagi"
	case h >= 11 && h < 15:
		return "siang"
	case h >= 15 && h < 18 || (h == 18 && now.Minute() < 30):
		return "sore"
	default:
		return "malam"
	}
}

// ComposeReminderText menghasilkan teks pengingat manis dari Persona Dara.
// Jika LLM error atau tidak tersedia, fallback template otomatis digunakan.
func (c *Composer) ComposeReminderText(ctx context.Context, taskText string) string {
	nowLocal := time.Now().In(c.timeZone)
	timeGreeting := getTimeGreeting(nowLocal)

	// Fallback jika LLM gagal atau tidak ada API Key
	fallbackText := fmt.Sprintf("%s bep, %s udah dikerjakan blm sayangg 💗", timeGreeting, strings.ToLower(taskText))

	if c.llmClient == nil {
		return fallbackText
	}

	personaGuide := c.loadPersonaContent()

	systemPrompt := fmt.Sprintf(`%s

PENTING — WAKTU SEKARANG:
Jam sekarang: %s WIB (Waktu: %s).
Instruksi: Tuliskan pengingat untuk tugas berikut ini kepada pasanganmu dengan gayamu yang natural, santai, dan menanyakan progres tugasnya. 
DILARANG kaku seperti alarm/kalender. Huruf kecil semua. Maksimal 1-2 kalimat pendek saja. Tidak perlu tanda kutip.`,
		personaGuide,
		nowLocal.Format("15:04"),
		timeGreeting,
	)

	userPrompt := fmt.Sprintf("Ingatkan tugas ini ke aku: %s", taskText)

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.llmClient.GenerateText(ctxTimeout, llm.PromptParams{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
	})
	if err != nil || strings.TrimSpace(resp) == "" {
		return fallbackText
	}

	// Bersihkan jika ada newline berlebih atau tanda kutip pembungkus
	clean := strings.Trim(strings.TrimSpace(resp), `"'`+"\n\r")
	clean = strings.ReplaceAll(clean, "\n", " ")
	return clean
}

// ComposeMorningBriefing menghasilkan briefing pagi yang manis merangkum agenda dan reminder hari ini.
func (c *Composer) ComposeMorningBriefing(ctx context.Context, agendaSummary string) string {
	fallbackText := "pagii sayanggg, semangat buat hari ini yaa 💗"
	if c.llmClient == nil {
		return fallbackText
	}

	personaGuide := c.loadPersonaContent()
	systemPrompt := fmt.Sprintf(`%s

Instruksi: Tulis sapaan pagi ("briefing pagi") yang hangat, santai, dan penuh semangat untuk pasanganmu. Rangkum secara singkat jadwal/tugasnya hari ini tanpa terlihat formal atau kaku. Maksimal 2 kalimat pendek. Huruf kecil semua.`, personaGuide)

	userPrompt := fmt.Sprintf("Ini agenda dan tugasku hari ini: %s", agendaSummary)

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.llmClient.GenerateText(ctxTimeout, llm.PromptParams{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
	})
	if err != nil || strings.TrimSpace(resp) == "" {
		return fallbackText
	}
	clean := strings.Trim(strings.TrimSpace(resp), `"'`+"\n\r")
	return strings.ReplaceAll(clean, "\n", " ")
}

// ComposeSayHi menghasilkan sapaan santai spontan tanpa topik sensitif.
func (c *Composer) ComposeSayHi(ctx context.Context) string {
	nowLocal := time.Now().In(c.timeZone)
	timeGreeting := getTimeGreeting(nowLocal)

	fallbacks := map[string]string{
		"pagi":  "pagii sayanggg, udh bangun blm 💗",
		"siang": "siang bep, lg sibuk yaa",
		"sore":  "sore sayangg, lg apa bep 💗",
		"malam": "malem sayangg, istirahat gihh bep",
	}
	fallbackText := fallbacks[timeGreeting]
	if fallbackText == "" {
		fallbackText = "sayangg lg apa nih 💗"
	}

	if c.llmClient == nil {
		return fallbackText
	}

	personaGuide := c.loadPersonaContent()
	systemPrompt := fmt.Sprintf(`%s

PENTING — WAKTU SEKARANG:
Jam sekarang: %s WIB (Waktu: %s).
Instruksi: Sapa pasanganmu secara spontan dan santai (say hi). Sesuaikan dengan jam waktu sekarang.
DILARANG membahas topik sensitif (uang, kesehatan, masalah keluarga, dll). Huruf kecil semua. Maksimal 1-2 bubble sangat pendek.`,
		personaGuide,
		nowLocal.Format("15:04"),
		timeGreeting,
	)

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.llmClient.GenerateText(ctxTimeout, llm.PromptParams{
		SystemPrompt: systemPrompt,
		UserPrompt:   "Sapa aku dong secara spontan",
		Temperature:  0.8,
	})
	if err != nil || strings.TrimSpace(resp) == "" {
		return fallbackText
	}
	clean := strings.Trim(strings.TrimSpace(resp), `"'`+"\n\r")
	return strings.ReplaceAll(clean, "\n", " ")
}

