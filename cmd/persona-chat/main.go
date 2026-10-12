package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"mini-assisted/internal/llm"
)

func loadPersonaGuide(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func getTimeContext() (string, string) {
	now := time.Now()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err == nil {
		now = now.In(loc)
	}
	timeStr := now.Format("15:04")
	hour := now.Hour()
	period := "Malam"
	if hour >= 5 && hour < 11 {
		period = "Pagi"
	} else if hour >= 11 && hour < 15 {
		period = "Siang"
	} else if hour >= 15 && hour < 18 {
		period = "Sore"
	} else if hour >= 18 && hour < 24 {
		period = "Malam"
	} else {
		period = "Tengah Malam"
	}
	return timeStr, period
}

func main() {
	personaPath := flag.String("persona", "persona/dara.md", "Path ke berkas style guide persona")
	personaName := flag.String("name", "Dara", "Nama persona yang ditampilkan")
	userName := flag.String("user", "Kamu", "Nama pengguna di terminal")
	flag.Parse()

	guideText, err := loadPersonaGuide(*personaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error membaca berkas persona %s: %v\n", *personaPath, err)
		os.Exit(1)
	}

	client := llm.NewClient(llm.Config{})

	systemPromptBase := fmt.Sprintf(`Kamu adalah asisten pribadi bernama %s yang berbicara meniru gaya persona berikut secara tepat.
Patuhi semua aturan struktur, panjang pesan, huruf kecil/kapital, tanda baca, panggilan khas, dan batasan negatif persona.
Gunakan bubble pesan yang sangat pendek dan natural layaknya chat WhatsApp santai.
JANGAN pernah bersikap formal seperti chatbot/customer service.
JANGAN pernah menyebut diri sebagai model bahasa atau AI.

%s`, *personaName, guideText)

	currentTimeStr, currentPeriod := getTimeContext()
	fmt.Println("=================================================================")
	fmt.Printf(" Mode Chat Interaktif CLI Persona: %s (v2)\n", *personaName)
	fmt.Printf(" Style Guide: %s\n", *personaPath)
	fmt.Printf(" Waktu Sekarang: %s WIB (%s)\n", currentTimeStr, currentPeriod)
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println(" Shortcut Perintah Simulasi (Tanpa Perlu Mancing):")
	fmt.Println("   'sayhi'              -> Sapaan pembuka (sesuai jam saat ini)")
	fmt.Println("   'makan'              -> Simulasi pengingat makan siang (12:30 WIB)")
	fmt.Println("   'tidur'              -> Simulasi pengingat tidur malam (23:15 WIB)")
	fmt.Println("   'minum'              -> Simulasi pengingat minum air / jeda santai")
	fmt.Println("   'task [nama tugas]'  -> Simulasi pengingat tugas (15 mnt lg)")
	fmt.Println("   'late [nama tugas]'  -> Simulasi pengingat tugas terlewat")
	fmt.Println("   'capek'              -> Simulasi kamu curhat lelah/lembur")
	fmt.Println("   'selesai [proyek]'   -> Simulasi kabar gembira tugas kelar")
	fmt.Println("   'help'               -> Tampilkan daftar perintah ini")
	fmt.Println("   'clear'              -> Reset riwayat percakapan")
	fmt.Println("   'keluar' / 'exit'    -> Selesai")
	fmt.Println("=================================================================")
	fmt.Println()

	var history []llm.ChatMessage
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("[%s] > ", *userName)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		lower := strings.ToLower(input)
		if lower == "exit" || lower == "quit" || lower == "keluar" {
			fmt.Println("\nSampai jumpa!")
			break
		}

		if lower == "clear" || lower == "reset" {
			history = nil
			fmt.Println("[Riwayat percakapan dibersihkan]")
			fmt.Println()
			continue
		}

		if lower == "help" || lower == "bantuan" {
			fmt.Println("\n--- Daftar Shortcut Simulasi ---")
			fmt.Println("  sayhi             : Sapaan pembuka sesuai waktu")
			fmt.Println("  makan             : Pengingat makan siang")
			fmt.Println("  tidur             : Pengingat tidur malam")
			fmt.Println("  minum             : Pengingat minum air")
			fmt.Println("  task <nama>       : Pengingat jadwal tugas (contoh: task review dokumen)")
			fmt.Println("  late <nama>       : Pengingat tugas terlewat (contoh: late kirim berkas)")
			fmt.Println("  capek             : Simulasi curhat lelah kerjaan")
			fmt.Println("  selesai <hal>     : Simulasi kabar gembira (contoh: selesai proyek web)")
			fmt.Println("--------------------------------")
			fmt.Println()
			continue
		}

		nowTime, nowPeriod := getTimeContext()
		userMsgText := input

		switch {
		case lower == "sayhi" || lower == "hi":
			userMsgText = fmt.Sprintf("(Sistem: Waktu saat ini jam %s WIB [%s]. Sapa pengguna sesuai waktu sekarang dan gayamu)", nowTime, nowPeriod)
			fmt.Printf("[Simulasi: Sapaan Pembuka jam %s WIB]\n", nowTime)

		case lower == "makan" || lower == "remind makan":
			userMsgText = "(Sistem: Waktu jam 12:30 WIB [Siang]. Kamu bertugas mengirim notifikasi pengingat makan siang untuk pengguna sesuai gaya personamu)"
			fmt.Println("[Simulasi: Pengingat Makan Siang (12:30 WIB)]")

		case lower == "tidur" || lower == "remind tidur":
			userMsgText = "(Sistem: Waktu jam 23:15 WIB [Malam]. Kamu bertugas mengirim notifikasi pengingat istirahat/tidur malam dan ucapan good night sesuai gaya personamu)"
			fmt.Println("[Simulasi: Pengingat Tidur Malam (23:15 WIB)]")

		case lower == "minum" || lower == "remind minum":
			userMsgText = "(Sistem: Kirimkan pesan perhatian singkat agar pengguna minum air putih dan istirahat sejenak sesuai gaya personamu)"
			fmt.Println("[Simulasi: Pengingat Minum Air Putih]")

		case strings.HasPrefix(lower, "task ") || strings.HasPrefix(lower, "remind task "):
			taskName := strings.TrimPrefix(lower, "remind task ")
			taskName = strings.TrimPrefix(taskName, "task ")
			taskName = strings.TrimSpace(taskName)
			if taskName == "" {
				taskName = "review dokumen"
			}
			userMsgText = fmt.Sprintf(`(Sistem: Ada tugas "%s" yang jatuh tempo 15 menit lagi. Ingatkan pengguna secara santai dan natural seperti pasangan yang peduli — JANGAN kaya notif kalender. Tanyakan progresnya atau ingatkan dengan lembut, contoh: "[nama tugas]nya udah dikerjakann?" atau "bentar lg batas waktunya loh, [nama tugas]nya udah beres blm 💗". Sebutkan nama tugasnya.)`, taskName)
			fmt.Printf("[Simulasi: Pengingat Tugas '%s' (15 mnt lagi, gaya natural)]\n", taskName)

		case lower == "task" || lower == "remind task":
			userMsgText = `(Sistem: Ada tugas "review dokumen" yang jatuh tempo 15 menit lagi. Ingatkan pengguna secara santai dan natural seperti pasangan yang peduli — JANGAN kaya notif kalender. Tanyakan progresnya, contoh: "review dokumennya udah dikerjakann?" atau "bentar lg batas waktunya loh bep 💗". Sebutkan nama tugasnya.)`
			fmt.Println("[Simulasi: Pengingat Tugas 'review dokumen' (15 mnt lagi, gaya natural)]")

		case strings.HasPrefix(lower, "late ") || strings.HasPrefix(lower, "remind late "):
			taskName := strings.TrimPrefix(lower, "remind late ")
			taskName = strings.TrimPrefix(taskName, "late ")
			taskName = strings.TrimSpace(taskName)
			if taskName == "" {
				taskName = "kirim berkas"
			}
			userMsgText = fmt.Sprintf(`(Sistem: Tugas "%s" sudah lewat batas waktu 1 jam lalu dan belum selesai. Ingatkan dengan santai dan mellow seperti pasangan yang khawatir, tanyakan apakah sudah selesai. Sebutkan nama tugasnya.)`, taskName)
			fmt.Printf("[Simulasi: Pengingat Tugas Terlewat '%s']\n", taskName)

		case lower == "late" || lower == "remind late":
			userMsgText = `(Sistem: Tugas "kirim berkas" sudah lewat batas waktu 1 jam lalu dan belum selesai. Ingatkan dengan santai dan mellow seperti pasangan yang khawatir, tanyakan apakah sudah selesai. Sebutkan nama tugasnya.)`
			fmt.Println("[Simulasi: Pengingat Tugas Terlewat 'kirim berkas']")

		case lower == "capek":
			userMsgText = "Capek bgt hari ini banyak kerjaan"

		case strings.HasPrefix(lower, "selesai"):
			hal := strings.TrimPrefix(input, "selesai")
			hal = strings.TrimPrefix(hal, "Selesai")
			hal = strings.TrimSpace(hal)
			if hal == "" {
				hal = "proyek"
			}
			userMsgText = fmt.Sprintf("Akhirnya %s kelar juga tadi!", hal)
		}

		history = append(history, llm.ChatMessage{
			Role: "user",
			Text: userMsgText,
		})

		activeSystemPrompt := fmt.Sprintf("%s\n\n[Waktu saat ini: Jam %s WIB (%s). Wajib sesuaikan sapaan dan referensi waktu dengan waktu ini!]", systemPromptBase, nowTime, nowPeriod)

		ctx := context.Background()
		fmt.Print("...")
		resp, err := client.Chat(ctx, llm.ChatParams{
			SystemPrompt: activeSystemPrompt,
			Messages:     history,
			Temperature:  0.7,
		})
		fmt.Print("\r   \r") // Hapus indikator loading

		if err != nil {
			fmt.Printf("\n[Error dari Gemini: %v]\n\n", err)
			// Hapus pesan user terakhir jika gagal agar history tetap konsisten
			history = history[:len(history)-1]
			continue
		}

		// Simpan balasan ke history
		history = append(history, llm.ChatMessage{
			Role: "model",
			Text: resp,
		})

		fmt.Printf("[%s]:\n", *personaName)
		lines := strings.Split(resp, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				fmt.Printf("  %s\n", trimmed)
			}
		}
		fmt.Println()
	}
}
