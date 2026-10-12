// cmd/app adalah entry point binary tunggal mini-assisted.
// Membaca konfigurasi dari .env, membuka database SQLite,
// dan menjalankan HTTP server.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"mini-assisted/internal/chat"
	"mini-assisted/internal/llm"
	"mini-assisted/internal/persona"
	"mini-assisted/internal/quickadd"
	"mini-assisted/internal/scheduler"
	"mini-assisted/internal/server"
	"mini-assisted/internal/store"
	"time"
)




func main() {
	loadDotEnv()

	dbPath := envOr("DB_PATH", "mini-assisted.db")
	addr := envOr("ADDR", ":8080")
	passwordHash := os.Getenv("OWNER_PASSWORD_HASH")
	if passwordHash == "" {
		log.Fatal("[FATAL] OWNER_PASSWORD_HASH belum diset di .env\n" +
			"Generate dengan: go run ./cmd/genhash")
	}

	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		log.Fatalf("[FATAL] buka database %q: %v", dbPath, err)
	}
	defer st.Close()
	log.Printf("[OK] database: %s", dbPath)

	// Inisialisasi LLM Client, Zona Waktu, dan Komponen AI
	llmClient := llm.NewClient(llm.Config{})
	tz, _ := time.LoadLocation(envOr("TZ", "Asia/Jakarta"))
	composer := persona.NewComposer(llmClient, "persona/dara.md", tz)
	qaParser := quickadd.NewParser(llmClient, tz)
	chatSvc := chat.NewService(st, llmClient, "persona/dara.md", tz)

	srv := server.New(st, server.Config{
		OwnerPasswordHash: passwordHash,
		Addr:              addr,
		VAPIDPublicKey:    os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:   os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:      envOr("VAPID_SUBJECT", "mailto:owner@localhost"),
		QuickAddParser:    qaParser,
		ChatService:       chatSvc,
	})


	// Jalankan Background Scheduler (pemeriksaan reminder setiap 30 detik, briefing pagi, say hi)
	bgScheduler := scheduler.New(st, srv.PushService(), composer, 30*time.Second)
	bgScheduler.Start()
	defer bgScheduler.Stop()


	log.Printf("[OK] server berjalan di http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("[FATAL] server: %v", err)
	}

}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadDotEnv memuat file .env ke environment variables.
func loadDotEnv() {
	for _, p := range []string{".env", "../.env"} {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
			}
		}
		return
	}
}

// Pastikan binary bisa dikompilasi — ini hanya placeholder agar go vet senang.
var _ = fmt.Sprintf
