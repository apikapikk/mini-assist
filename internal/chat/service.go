// Package chat mengelola interaksi obrolan dua arah dengan Persona Dara menggunakan arsitektur Konteks Selektif.
package chat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"mini-assisted/internal/llm"
	"mini-assisted/internal/store"
)

// Service mengelola obrolan dan manipulasi konteks hemat token.
type Service struct {
	store       store.Store
	llmClient   *llm.Client
	personaPath string
	timeZone    *time.Location
}

// NewService membuat instance Service baru.
func NewService(st store.Store, client *llm.Client, personaPath string, tz *time.Location) *Service {
	if tz == nil {
		tz, _ = time.LoadLocation("Asia/Jakarta")
		if tz == nil {
			tz = time.Local
		}
	}
	if personaPath == "" {
		personaPath = "persona/dara.md"
	}
	return &Service{
		store:       st,
		llmClient:   client,
		personaPath: personaPath,
		timeZone:    tz,
	}
}

func (s *Service) loadPersonaContent() string {
	for _, p := range []string{s.personaPath, "../" + s.personaPath, "../../" + s.personaPath} {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return "Kamu adalah Dara, pasangan yang akrab, perhatian, santai, memanggil 'sayangg' atau 'bep', huruf kecil semua tanpa tanda titik formal."
}

// Reply memproses pesan pengguna, menyusun Konteks Selektif, memanggil LLM, dan menyimpan riwayat percakapan.
func (s *Service) Reply(ctx context.Context, userText string) (string, error) {
	nowLocal := time.Now().In(s.timeZone)

	// 1. Simpan pesan pengguna ke database SQLite
	_, _ = s.store.SaveMessage(store.Message{
		Role:      "user",
		Content:   userText,
		CreatedAt: nowLocal.UTC(),
	})

	if s.llmClient == nil {
		fallback := "iyaaa sayanggg, ada apa bep 💗"
		_, _ = s.store.SaveMessage(store.Message{
			Role:      "model",
			Content:   fallback,
			CreatedAt: time.Now().UTC(),
		})
		return fallback, nil
	}

	// 2. Ambil Konteks Selektif:
	// a. Fakta penting tentang pemilik (facts)
	facts, _ := s.store.ListFacts()
	var factsStr string
	if len(facts) > 0 {
		var fList []string
		for _, f := range facts {
			fList = append(fList, fmt.Sprintf("- %s: %s", f.Key, f.Value))
		}
		factsStr = "\nFakta tentang pemilik:\n" + strings.Join(fList, "\n")
	}

	// b. Rolling Summary obrolan terdahulu (chat_summaries)
	summary, hasSummary, _ := s.store.LatestChatSummary()
	var summaryStr string
	if hasSummary && summary.Summary != "" {
		summaryStr = "\nRingkasan obrolan sebelumnya:\n" + summary.Summary
	}

	// c. Ambil HANYA 10 pesan terakhir saja (hemat token)
	recentMsgs, _ := s.store.ListRecentMessages(10)
	var chatHistory []llm.ChatMessage
	for _, m := range recentMsgs {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		chatHistory = append(chatHistory, llm.ChatMessage{
			Role: role,
			Text: m.Content,
		})
	}

	// 3. Susun System Prompt Cerdas
	personaGuide := s.loadPersonaContent()
	systemPrompt := fmt.Sprintf(`%s
%s%s

PENTING — WAKTU SEKARANG:
Jam saat ini: %s WIB (Tanggal: %s).
Respon obrolan pasanganmu dengan kepribadian dan gayamu yang natural, huruf kecil semua.`,
		personaGuide,
		factsStr,
		summaryStr,
		nowLocal.Format("15:04"),
		nowLocal.Format("Monday, 02 Jan 2006"),
	)

	// 4. Panggil LLM Gemini
	ctxTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	reply, err := s.llmClient.Chat(ctxTimeout, llm.ChatParams{
		SystemPrompt: systemPrompt,
		Messages:     chatHistory,
		Temperature:  0.8,
	})
	if err != nil || strings.TrimSpace(reply) == "" {
		reply = "iyaaa bep, maaf td sempet putus jaringannya 💗"
	}

	cleanReply := strings.TrimSpace(reply)

	// 5. Simpan balasan persona ke DB
	_, _ = s.store.SaveMessage(store.Message{
		Role:      "model",
		Content:   cleanReply,
		CreatedAt: time.Now().UTC(),
	})

	// 6. Jalankan pembersihan / rolling summary jika pesan sudah menumpuk
	go s.maybeRollSummary()

	return cleanReply, nil
}

// maybeRollSummary memeriksa jika riwayat pesan melebihi batas (misal > 30 pesan),
// buat ringkasan baru lalu simpan ke chat_summaries.
func (s *Service) maybeRollSummary() {
	msgs, err := s.store.ListRecentMessages(40)
	if err != nil || len(msgs) < 30 {
		return
	}

	// Jika sudah banyak pesan, rangkum 20 pesan tertua
	var convoText strings.Builder
	for i := 0; i < 20; i++ {
		convoText.WriteString(fmt.Sprintf("%s: %s\n", msgs[i].Role, msgs[i].Content))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	prompt := fmt.Sprintf("Buatkan ringkasan obrolan santai ini dalam 2-3 kalimat padat tanpa menghilangkan konteks penting:\n\n%s", convoText.String())
	summaryResp, err := s.llmClient.GenerateText(ctx, llm.PromptParams{
		SystemPrompt: "Kamu adalah asisten perangkum obrolan.",
		UserPrompt:   prompt,
		Temperature:  0.2,
	})
	if err != nil || strings.TrimSpace(summaryResp) == "" {
		return
	}

	cutoff := msgs[19].CreatedAt
	_, _ = s.store.SaveChatSummary(store.ChatSummary{
		Summary:     strings.TrimSpace(summaryResp),
		CoversUntil: cutoff,
		CreatedAt:   time.Now().UTC(),
	})

	// Hapus pesan-pesan yang sudah dirangkum agar SQLite tetap ramping
	_ = s.store.DeleteMessagesBefore(cutoff)
}
