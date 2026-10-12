# TODO

Legenda: `[ ]` belum, `[~]` sedang berjalan, `[x]` selesai. Tugas khusus pemilik ditandai **(pemilik)**.

## Fase 0 — Keputusan & persiapan
- [ ] **(pemilik)** Jenis HP: Android atau iPhone (menentukan keandalan push)
- [ ] **(pemilik)** Pilih jalur TLS: domain/subdomain + Caddy, atau Tailscale HTTPS
- [ ] **(pemilik)** Cek akses VPS (SSH), versi Go, sisa RAM/disk
- [ ] **(pemilik)** Siapkan API key LLM dan set batas pengeluaran bulanan
- [x] Buat repo, `.gitignore` (`.env`, `*.db`, `raw/`), `.env.example`
- [x] Buat pasangan kunci VAPID (`cmd/genvapid`)

## Fase 1 — Persona jalan dulu ✅ SELESAI
Semua lokal. Persona Dara v3 lolos evaluasi (rata-rata 4.0, semua skenario ≥ 4).
- [x] **(pemilik)** Ekspor arsip chat ke folder lokal `raw/`
- [x] Parser: arsip → `[waktu, pengirim, teks]` (`cmd/chat-parse`, `internal/chatlog`)
- [x] Filter otomatis: PII & topik sensitif (`cmd/chat-filter`, `internal/chatfilter`)
- [x] **(pemilik)** Tinjau manual hasil filter (`output/review.jsonl`)
- [x] Statistik gaya agregat (`cmd/chat-stats`, `internal/chatstats`)
- [x] Tulis `persona/netral.md` dan `persona/dara.md` (v1 → v3 final)
- [x] `cmd/persona-test`: CLI skenario + tabel markdown
- [x] `cmd/persona-chat`: CLI chat interaktif multi-turn
- [x] Tulis `persona/eval/scenarios.md` (8 skenario, tanpa topik sensitif)
- [x] **(pemilik)** Evaluasi v1 (3.38) → v2 (3.75) → v3 (4.0) — **LOLOS**
- [ ] **(pemilik)** Hapus atau arsipkan data mentah secara offline setelah style guide final

## Fase 2 — Kerangka PWA ✅ SELESAI
Server Go lokal + SQLite + PWA Shell + Embedded static files.

### Sprint 2.1 — DB + Store Layer ✅ SELESAI
- [x] Tambah dependency: `modernc.org/sqlite` (pure-Go, tanpa CGO)
- [x] `internal/store`: interface + implementasi SQLite
- [x] Skema & migrasi otomatis saat server start:
  - `reminders(id, text, due_at_utc, repeat_rule, status, created_at)`
  - `events(id, title, start_at_utc, end_at_utc, notes)`
  - `facts(id, key, value, updated_at)` ← konteks LLM, disimpan selektif
  - `push_subscriptions(id, endpoint, p256dh, auth, created_at)`
  - `messages(id, role, content, created_at)` ← untuk chat Fase 6
  - `chat_summaries(id, summary, covers_until, created_at)` ← rolling summary LLM
- [x] Unit test store CRUD (5 test, semua PASS)

### Sprint 2.2 — HTTP Server + Auth ✅ SELESAI
- [x] `internal/server`: HTTP server `net/http`, routing, middleware
- [x] Login pemilik: hash password (bcrypt), cookie sesi, rate limiting (5x/15 menit per IP)
- [x] Rute dasar: `POST /login`, `GET /logout`, middleware auth untuk semua rute
- [x] `cmd/app`: entry point binary tunggal, baca config dari `.env`
- [x] `cmd/genhash`: alat bantu generate bcrypt hash untuk setup password

### Sprint 2.3 — PWA Shell ✅ SELESAI
- [x] `manifest.json`: nama, ikon (192 & 512 SVG maskable), warna tema, `display: standalone`
- [x] Service Worker (`sw.js`): cache aset offline, penanganan lifecycle & push notification
- [x] Halaman Hari Ini (`web/templates/index.html`): mobile-first layout, tanggal lokal, quick-add form, daftar reminder aktif
- [x] Halaman Login (`web/templates/login.html`): antarmuka login minimalis & responsif
- [x] CSS & JS vanilla modern tanpa dependensi/build step (`web/static/css/style.css`, `web/static/js/app.js`)

### Sprint 2.4 — Embed & Build ✅ SELESAI
- [x] Embed semua frontend dengan `go:embed` di package `web/web.go`
- [x] Integrasi aset statis, template, dan endpoint API pengingat di `internal/server/server.go`
- [x] `go build` menghasilkan 1 binary statis mandiri tanpa file eksternal (Build verified)

### Deploy (tugas pemilik)
- [ ] **(pemilik)** Setup domain/subdomain & Caddy di VPS
- [ ] **(pemilik)** Buat unit systemd untuk binary
- [ ] **(pemilik)** Verifikasi HTTPS + install PWA di HP

## Fase 3 — Push Notifikasi ✅ SELESAI (Sisi Kode)
- [x] `internal/push`: modul kirim Web Push berotentikasi VAPID (`github.com/SherClockHolmes/webpush-go`)
- [x] Endpoint langganan: `GET /api/push/vapid-public-key`, `POST /api/push/subscribe`, simpan ke SQLite
- [x] Halaman Pengaturan (`/settings`) dengan tombol "Aktifkan Notifikasi di HP Ini"
- [x] Endpoint uji: `POST /api/push/test` (broadcast ke semua perangkat pemilik)
- [x] Tangani subscription kedaluwarsa otomatis (HTTP 404/410 langsung dihapus dari DB)
- [ ] **(pemilik)** Verifikasi tes push langsung di HP saat HTTPS aktif di VPS/laptop

## Fase 4 — Reminder & Todo-Task Manager ✅ SELESAI
- [x] API CRUD & Task reminder:
  - `GET /api/reminders` (filter status `pending` / `done`)
  - `POST /api/reminders` (dukung custom `due_at` & `repeat_rule`)
  - `PUT /api/reminders/{id}` (edit teks, due date, atau aturan pengulangan)
  - `POST /api/reminders/{id}/done` & `POST /api/reminders/{id}/undone` (toggle selesai / batalkan)
  - `DELETE /api/reminders/{id}`
- [x] `internal/scheduler`: background tick setiap 30 detik (`scheduler.New()`, `Start()`, `Stop()`)
- [x] Peracik teks persona di `internal/persona/composer.go`: meracik pesan pengingat peduli (Dara v3 via Gemini LLM)
- [x] Cadangan template teks jika LLM gagal / offline
- [x] Aturan berulang otomatis fleksibel:
  - `daily` (setiap hari)
  - `weekdays` (Senin-Jumat)
  - `weekly` (seminggu sekali)
  - `days:1,3,5` (hari tertentu dalam seminggu, misal Sen, Rab, Jum)
  - `interval:N` (setiap N hari sekali)
  - Otomatis menghitung dan menjadwalkan tanggal jatuh tempo berikutnya
- [x] Otomatis kirim Web Push saat reminder jatuh tempo ke HP pemilik
- [x] Antarmuka Todo List di PWA Shell:
  - Tab filter **Aktif** vs **Selesai**
  - Modal form detail untuk atur tanggal, jam, dan hari pengulangan
  - Checkbox toggle selesai, coret teks task done, dan tombol hapus
- [x] Unit test scheduler tick & recurring rule (`internal/scheduler/scheduler_test.go`, `internal/scheduler/repeat_test.go`) lolos 100%

## Fase 5 — Tahu jadwal & say hi ✅ SELESAI
- [x] CRUD events API: `GET/POST /api/events`, `DELETE /api/events/{id}`
- [x] Briefing pagi terotomatisasi (events + reminder hari ini) di jam 07:00-08:30 WIB
- [x] Kotak quick-add cerdas (`internal/quickadd/parser.go`): teks bahasa natural → LLM parse → konfirmasi (`POST /api/quick-add/confirm`) → simpan
- [x] Say hi berkala: maksimal 2x per hari, jeda min 4 jam, mematuhi jam tenang (22:00 - 08:00 WIB), tanpa topik sensitif
- [x] Unit test events & quick-add endpoints di `internal/server/server_test.go` lolos 100%

## Fase 6 — Chat ✅ SELESAI
- [x] Layar chat interaktif di PWA (`/chat` & `web/templates/chat.html`) dengan balon obrolan responsif
- [x] API obrolan: `POST /api/chat`, `GET /api/chat/messages`
- [x] Konteks Selektif hemat token (`internal/chat/service.go`):
  - Injeksi Persona Dara v3
  - Injeksi waktu real-time Asia/Jakarta
  - Injeksi fakta penting pemilik (`facts`)
  - Injeksi ringkasan rolling (`chat_summaries`)
  - Mengirim HANYA 10 pesan terakhir saja (tidak mengirim seluruh database)
- [x] Rolling summary worker: otomatis merangkum obrolan lama jika > 30 pesan dan membersihkan riwayat lama
- [x] Unit test obrolan di `internal/server/server_test.go` lolos 100%

## Deployment & Verifikasi
- [ ] **Langkah 1 (Lokal):** Uji coba langsung di localhost (Browser + Chat + Reminder)
- [ ] **Langkah 2 (VPS):** Deploy binary tunggal ke VPS + Caddy TLS + Systemd (oleh pemilik)


## Nanti / ide
- [ ] Impor dari Google Calendar / `.ics`
- [ ] Quick-add lewat suara
- [ ] ntfy sebagai kanal push cadangan
- [ ] Backup SQLite otomatis

## Catatan keputusan
- Tujuan utama: persona recreating yang terbukti jalan; fitur lain mengikuti
- Antarmuka: PWA + Web Push (Telegram ditinggalkan: registrasi terlalu ribet)
- DB: SQLite (bukan Firebase) — single user, gratis, offline, tanpa vendor lock-in
- PWA local-first: IndexedDB di HP untuk cache, sync ke VPS SQLite saat online
- Context LLM selektif: facts + rolling summary + N pesan terakhir (bukan semua history)
- Go + SQLite (`modernc.org/sqlite`, pure-Go, tanpa CGO) + Caddy
- Tanpa fine-tuning; persona lewat prompt, konteks lewat `facts` + `chat_summaries`
- Dokumentasi berbahasa Indonesia
