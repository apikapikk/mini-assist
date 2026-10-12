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

type Scenario struct {
	Number  int
	Title   string
	Context string
	Trigger string
}

func parseScenarios(path string) ([]Scenario, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var scenarios []Scenario
	var current *Scenario

	scanner := bufio.NewScanner(f)
	num := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "### Skenario ") {
			if current != nil {
				scenarios = append(scenarios, *current)
			}
			num++
			title := strings.TrimPrefix(line, "### Skenario ")
			colonIdx := strings.Index(title, ":")
			if colonIdx != -1 {
				title = strings.TrimSpace(title[colonIdx+1:])
			}
			current = &Scenario{
				Number: num,
				Title:  title,
			}
			continue
		}
		if current != nil {
			if strings.HasPrefix(line, "- **Konteks:**") {
				current.Context = strings.TrimSpace(strings.TrimPrefix(line, "- **Konteks:**"))
			} else if strings.HasPrefix(line, "- **Pemicu:**") {
				current.Trigger = strings.TrimSpace(strings.TrimPrefix(line, "- **Pemicu:**"))
			}
		}
	}
	if current != nil {
		scenarios = append(scenarios, *current)
	}
	return scenarios, scanner.Err()
}

func loadPersonaGuide(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func mockResponse(personaName string, sc Scenario) string {
	lower := strings.ToLower(personaName)
	if strings.Contains(lower, "dara") {
		switch sc.Number {
		case 1:
			return "pagii sayanggg, udh bangun blm?"
		case 2:
			return "sayangg jgn lupa makan yaa"
		case 3:
			return "bep 15 mnt lg jgn lupa review dokumen yaa"
		case 4:
			return "km capek yaa.. istirahat dulu gih sayangg"
		case 5:
			return "sayangg kirim berkasnya udh blm yaa"
		case 6:
			return "km lg apa bep"
		case 7:
			return "ihh hebat bgt sayanggg 💗"
		case 8:
			return "tidur gihh, good night sayanggg 💗"
		default:
			return "iyaaa sayanggg"
		}
	}

	// Netral fallback
	switch sc.Number {
	case 1:
		return "Pagi! Ini agenda dan pengingat kamu untuk hari ini ya. Semangat!"
	case 2:
		return "Pengingat: Waktunya makan siang yaa. Jangan sampai terlewat!"
	case 3:
		return "Pengingat: 15 menit lagi ada jadwal review dokumen ya."
	case 4:
		return "Kerja bagus hari ini! Kalau capek, ambil istirahat sebentar ya."
	case 5:
		return "Pengingat: Tugas kirim berkas sudah melewati batas waktu 1 jam lalu."
	case 6:
		return "Sore! Lagi santai atau masih sibuk nih?"
	case 7:
		return "Wah selamat ya! Keren banget akhirnya selesai."
	case 8:
		return "Hari ini udah selesai, selamat istirahat yang cukup ya."
	default:
		return "Siap, dicatat ya."
	}
}

func generateForScenario(ctx context.Context, client *llm.Client, personaText string, personaPath string, sc Scenario, mock bool) (string, error) {
	if mock || os.Getenv("LLM_API_KEY") == "" && os.Getenv("GEMINI_API_KEY") == "" {
		return mockResponse(personaPath, sc), nil
	}

	sysPrompt := fmt.Sprintf(`Kamu adalah asisten pribadi yang berbicara meniru gaya persona berikut secara tepat.
Patuhi semua aturan struktur, panjang pesan, huruf kecil/kapital, tanda baca, panggilan khas, dan batasan negatif persona.
PENTING:
1. Perhatikan waktu/jam pada situasi dan sesuaikan sapaan secara akurat (pagi/siang/sore/malam).
2. Jika ada nama tugas atau agenda, sebutkan nama tugasnya secara spesifik — tapi JANGAN kaku seperti notif kalender ("15 mnt lg ada jadwal X"). Ingatkan secara natural seperti pasangan yang peduli: tanyakan progresnya ("review dokumennya udah dikerjakann?") atau ingatkan lembut ("bentar lg batas waktunya loh bep 💗").
3. HANYA hasilkan teks pesan singkat yang dikirim ke pengguna tanpa penjelasan pembuka/penutup.

%s`, personaText)

	userPrompt := fmt.Sprintf(`Situasi: %s
Pemicu aksi: %s

Tuliskan pesan notifikasi/chat sesuai persona:`, sc.Context, sc.Trigger)

	return client.GenerateText(ctx, llm.PromptParams{
		SystemPrompt: sysPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
	})
}

func main() {
	persona1 := flag.String("persona", "persona/dara.md", "Path ke berkas style guide persona utama")
	persona2 := flag.String("persona2", "", "Path ke berkas style guide persona kedua untuk perbandingan (opsional)")
	scenariosPath := flag.String("scenarios", "persona/eval/scenarios.md", "Path ke berkas skenario evaluasi")
	repeatCount := flag.Int("n", 1, "Jumlah pengulangan evaluasi per skenario")
	mock := flag.Bool("mock", false, "Gunakan mode mock tanpa memanggil API LLM")
	flag.Parse()

	scenarios, err := parseScenarios(*scenariosPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error membaca skenario %s: %v\n", *scenariosPath, err)
		os.Exit(1)
	}

	guide1, err := loadPersonaGuide(*persona1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error membaca persona %s: %v\n", *persona1, err)
		os.Exit(1)
	}

	var guide2 string
	if *persona2 != "" {
		g, err := loadPersonaGuide(*persona2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error membaca persona2 %s: %v\n", *persona2, err)
			os.Exit(1)
		}
		guide2 = g
	}

	client := llm.NewClient(llm.Config{})
	ctx := context.Background()

	// Cetak Tabel Markdown yang siap ditempel ke persona/eval/nilai.md
	fmt.Println("### Hasil Pengujian Persona CLI")
	fmt.Printf("Model: Gemini | Persona 1: %s", *persona1)
	if *persona2 != "" {
		fmt.Printf(" | Persona 2: %s", *persona2)
	}
	fmt.Println()

	if *persona2 != "" {
		fmt.Println("| No | Skenario | Respon Persona 1 | Respon Persona 2 | Nilai (1-5) | Catatan |")
		fmt.Println("|---|---|---|---|---|---|")
	} else {
		fmt.Println("| No | Skenario | Respon Persona | Nilai (1-5) | Catatan Pemilik |")
		fmt.Println("|---|---|---|---|---|")
	}

	for _, sc := range scenarios {
		for i := 0; i < *repeatCount; i++ {
			resp1, err := generateForScenario(ctx, client, guide1, *persona1, sc, *mock)
			if err != nil {
				resp1 = fmt.Sprintf("[Error: %v]", err)
			}
			// Escape pipe char
			resp1 = strings.ReplaceAll(resp1, "|", "/")
			resp1 = strings.ReplaceAll(resp1, "\n", " <br> ")

			if *persona2 != "" {
				resp2, err := generateForScenario(ctx, client, guide2, *persona2, sc, *mock)
				if err != nil {
					resp2 = fmt.Sprintf("[Error: %v]", err)
				}
				resp2 = strings.ReplaceAll(resp2, "|", "/")
				resp2 = strings.ReplaceAll(resp2, "\n", " <br> ")

				fmt.Printf("| %d | %s | %s | %s | | |\n", sc.Number, sc.Title, resp1, resp2)
			} else {
				fmt.Printf("| %d | %s | %s | | |\n", sc.Number, sc.Title, resp1)
			}
			if !*mock {
				time.Sleep(2 * time.Second)
			}
		}
	}
}
