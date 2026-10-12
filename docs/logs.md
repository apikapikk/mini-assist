# Development Logs

Dokumen ini melacak progres pengerjaan proyek asisten pribadi berbasis Go & PWA sesuai dengan panduan di `docs/AGENTS.md`, `docs/SCOPE.md`, dan `docs/TODO.md`.

---

## Log Aktivitas

### 2026-10-11
- **Status & Selesai Dikerjakan:**
  - Membaca dan memahami dokumen `docs/AGENTS.md`, `docs/SCOPE.md`, dan `docs/TODO.md`.
  - Membuat berkas pelacak aktivitas `docs/logs.md`.
  - Menginisialisasi Git repository (`git init`).
  - Menginisialisasi Go module `mini-assisted` (`go mod init mini-assisted`).
  - Menyiapkan berkas `.gitignore` (`.env`, `*.db`, `raw/`, `output/`, build artifacts).
  - Menyiapkan berkas `.env.example`.
  - **Tugas A1 (Parser WhatsApp Offline):**
    - Menyediakan dataset sintetis di `testdata/sample_chat.txt`.
    - Membangun paket `internal/chatlog` untuk parsing chat multi-baris, penanda media, dan pesan sistem.
    - Membuat CLI `cmd/chat-parse` (`-in`, `-out`, `-sender`).
    - Unit test table-driven di `internal/chatlog/parser_test.go` lolos 100%.
  - **Tugas A2 (Filter PII & Topik Sensitif):**
    - Membuat daftar kata kunci topik sensitif di `config/sensitif.txt`.
    - Membangun paket `internal/chatfilter` untuk sanitasi URL, Email, Nomor Telepon, @mention, dan blocklist entitas.
    - Membuat CLI `cmd/chat-filter` (`-in`, `-out`, `-sensitif`, `-blocklist`, `-review`).
    - Laporan ringkas sesuai spek: angka masuk, diterima, dibuang, dan diragukan.
    - Unit test table-driven di `internal/chatfilter/filter_test.go` lolos 100%.
  - **Tugas A3 (Statistik Gaya Agregat):**
    - Membangun paket `internal/chatstats` untuk kalkulasi panjang pesan, median, casing, tanda baca, pemanjangan huruf ("iyaaa", "sayanggg"), emoji, kata pembuka, penutup, dan filler.
    - Membuat CLI `cmd/chat-stats` (`-in`, `-out`, `-min-count`, `-sender`).
    - Menjamin privasi dengan ambang batas minimal kemunculan token (tanpa kutipan pesan utuh).
    - Unit test di `internal/chatstats/stats_test.go` lolos 100%.
  - **Eksekusi Pipeline pada Data Mentah (`raw/daraaaaaaabebb_persona.txt`):**
    - Berhasil mem-parse 1.372 pesan milik `daraaaaaaabebb` secara offline (0 token LLM terpakai).
    - Berhasil memfilter pesan (1.365 diterima, 7 dibuang karena topik sensitif, 2 diragukan).
    - Menghasilkan ringkasan agregat di `output/dara_stats.json`.
  - **Tugas A4 (Style Guide Persona):**
    - Membuat template panduan gaya di `persona/_template.md`.
    - Membuat persona cadangan di `persona/netral.md`.
    - Menulis style guide `persona/dara.md` v1 dengan contoh kalimat sintetis baru berdasarkan statistik agregat (tanpa membocorkan isi chat privat).
  - **Tugas A5 (Pengujian Persona CLI & Evaluasi):**
    - Membuat daftar 8 skenario pengujian di `persona/eval/scenarios.md`.
    - Menyiapkan lembar penilaian di `persona/eval/nilai.md`.
    - Membangun modul client Google Gemini di `internal/llm` (pure Go standard library, tanpa dependensi pihak ketiga).
    - Membangun CLI `cmd/persona-test` (`-persona`, `-persona2`, `-scenarios`, `-n`, `-mock`) dengan keluaran tabel Markdown siap pakai.
  - **Tugas 1B (Mode Chat Interaktif CLI):**
    - Menambahkan dukungan obrolan multi-turn (`internal/llm.Chat`, `ChatMessage`, `ChatParams`).
    - Membuat CLI interaktif di [`cmd/persona-chat`](file:///home/fpans/Projects/mini-assisted/cmd/persona-chat/main.go) (`-persona`, `-name`, `-user`).
    - Fitur interaktif: perintah `sayhi` (minta persona menyapa duluan), `clear` (reset memori), dan `keluar` / `exit`.
    - Berhasil diuji coba live mengobrol dua arah dengan Persona Dara menggunakan model Gemini.
  - **Iterasi Persona Dara v2 (Berdasarkan Evaluasi Pemilik):**
    - Evaluasi v1 memperoleh skor 3.38 dengan masukan utama: *pembacaan waktu kurang akurat* dan *belum menyebutkan nama tugas secara spesifik*.
    - Memperbarui style guide ke [`persona/dara.md`](file:///home/fpans/Projects/mini-assisted/persona/dara.md) (v2) dengan aturan ketat pembagian waktu (pagi, siang, sore, malam) dan kewajiban menyebutkan nama tugas dalam reminder.
    - Menambahkan injeksi konteks waktu jam real-time Asia/Jakarta ke prompt [`cmd/persona-chat`](file:///home/fpans/Projects/mini-assisted/cmd/persona-chat/main.go).
    - Memperbarui evaluasi pengujian di [`persona/eval/nilai.md`](file:///home/fpans/Projects/mini-assisted/persona/eval/nilai.md) untuk versi v2.
- **Status Fase 1:** ✅ **SELESAI & LOLOS** — Persona Dara v3 rata-rata 4.0, semua skenario bernilai 4, disetujui pemilik.

---

### 2026-10-12
- **Keputusan Arsitektur (diskusi dengan pemilik):**
  - DB tetap **SQLite** (bukan Firebase): single user, gratis, offline, tanpa vendor lock-in, pure-Go driver (`modernc.org/sqlite`, tanpa CGO).
  - PWA **local-first**: IndexedDB di HP untuk cache lokal, sync ke VPS SQLite saat online. HP tidak perlu selalu fetch dari server.
  - **Context LLM selektif**: LLM tidak membaca semua history. Strategi: `facts` table (konteks statis pemilik) + `chat_summaries` (rolling summary percakapan) + N pesan terakhir saja. Hemat token, performa lebih baik.
  - Fase 6 (Chat) tetap ada di rencana, namun dikerjakan setelah Fase 4 (Reminder) stabil.
- **Pembaruan dokumen:**
  - `docs/TODO.md` diperbarui: Fase 1 ditandai selesai, sprint Fase 2 dirinci (2.1–2.4), catatan keputusan diperbarui.
  - `docs/logs.md` diperbarui (entri ini).
- **Akan Dikerjakan (Next Up):**
  - **Sprint 2.1** — DB + Store Layer: tambah `modernc.org/sqlite`, buat `internal/store`, skema & migrasi otomatis, unit test.
  - **Sprint 2.2** — HTTP server + session/login.

---

### 2026-10-12 (lanjutan — Sprint 2.1)
- **Sprint 2.1 — DB + Store Layer: ✅ SELESAI**
  - Menambahkan dependency `modernc.org/sqlite v1.60.1` (pure-Go, tanpa CGO) ke `go.mod`.
  - Membuat [`internal/store/store.go`](file:///home/fpans/Projects/mini-assisted/internal/store/store.go): interface `Store` + semua model data (`Reminder`, `Event`, `Fact`, `PushSubscription`, `Message`, `ChatSummary`).
  - Membuat [`internal/store/sqlite.go`](file:///home/fpans/Projects/mini-assisted/internal/store/sqlite.go): implementasi SQLite dengan `OpenSQLite()` + migrasi otomatis saat start. Semua tabel dibuat via `CREATE TABLE IF NOT EXISTS`. WAL mode aktif.
  - Skema tabel: `reminders`, `events`, `facts` (upsert by key), `push_subscriptions` (upsert by endpoint), `messages`, `chat_summaries`.
  - `facts` dan `chat_summaries` dirancang untuk context LLM selektif — LLM hanya baca facts relevan + rolling summary + N pesan terakhir, bukan semua history.
  - Unit test [`internal/store/sqlite_test.go`](file:///home/fpans/Projects/mini-assisted/internal/store/sqlite_test.go): 5 test (Reminder, Event, Fact, Message+Summary, PushSubscription) — semua **PASS** (`ok mini-assisted/internal/store 0.039s`).
- **Akan Dikerjakan (Next Up):**
  - **Sprint 2.2** — HTTP server + auth: `internal/server`, login (bcrypt + cookie sesi), rate limiting, `cmd/app` binary entry point.

---

### 2026-10-12 (lanjutan — Sprint 2.2)
- **Sprint 2.2 — HTTP Server + Auth: ✅ SELESAI**
  - Menambahkan dependency `golang.org/x/crypto v0.58.0` untuk bcrypt.
  - Membuat [`internal/server/server.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server.go): struct `Server`, `Config`, routing `net/http`, helper `writeJSON`/`writeError`.
  - Membuat [`internal/server/auth.go`](file:///home/fpans/Projects/mini-assisted/internal/server/auth.go): `authManager` (sesi in-memory + rate limiting), handler `POST /login` & `GET /logout`, middleware `requireAuth`.
    - Sesi 30 hari, token acak 32-byte hex.
    - Rate limit: maks 5 percobaan gagal per 15 menit per IP (mempertimbangkan header Caddy `X-Real-IP`).
    - Login via JSON `{"password":"..."}` atau form-data POST.
  - Membuat [`cmd/app/main.go`](file:///home/fpans/Projects/mini-assisted/cmd/app/main.go): entry point binary tunggal — load `.env`, buka SQLite, start HTTP server.
  - Membuat [`cmd/genhash/main.go`](file:///home/fpans/Projects/mini-assisted/cmd/genhash/main.go): alat bantu generate bcrypt hash untuk set `OWNER_PASSWORD_HASH` di `.env`.
  - Unit test [`internal/server/server_test.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server_test.go): 7 test (login sukses, password salah, rate limit, auth middleware, redirect browser, logout) — semua **PASS** (`ok mini-assisted/internal/server 0.993s`).
  - Build `cmd/app` dan `cmd/genhash` berhasil.
- **Akan Dikerjakan (Next Up):**
  - **Sprint 2.3 & 2.4** — PWA Shell & Static Embed: `manifest.json`, `sw.js`, HTML/CSS mobile-first, integrasi embed ke Go binary.

---

### 2026-10-12 (lanjutan — Sprint 2.3 & 2.4)
- **Sprint 2.3 & 2.4 — PWA Shell & Single Binary Embed: ✅ SELESAI**
  - Membuat Web App Manifest di [`web/static/manifest.json`](file:///home/fpans/Projects/mini-assisted/web/static/manifest.json) dengan `display: standalone`, tema gelap (`#0f172a`), dan ikon maskable SVG 192px/512px.
  - Membuat Service Worker di [`web/static/sw.js`](file:///home/fpans/Projects/mini-assisted/web/static/sw.js) untuk caching offline dan penanganan event `push` / `notificationclick`.
  - Mendesain antarmuka mobile-first responsif:
    - CSS modern vanilla: [`web/static/css/style.css`](file:///home/fpans/Projects/mini-assisted/web/static/css/style.css).
    - Logika client JS (registrasi SW, fetch reminder, mark done): [`web/static/js/app.js`](file:///home/fpans/Projects/mini-assisted/web/static/js/app.js).
    - Shell utama PWA: [`web/templates/index.html`](file:///home/fpans/Projects/mini-assisted/web/templates/index.html).
    - Halaman login pemilik: [`web/templates/login.html`](file:///home/fpans/Projects/mini-assisted/web/templates/login.html).
  - Meng-embed semua frontend via `go:embed` di [`web/web.go`](file:///home/fpans/Projects/mini-assisted/web/web.go).
  - Menghubungkan rute web dan API dasar di [`internal/server/server.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server.go) (`/`, `/login`, `/manifest.json`, `/sw.js`, `/static/*`, `/api/reminders`).
  - Menjalankan uji menyeluruh (`go test ./...`) dan build binary statis (`go build ./cmd/app`): **100% PASS & Berhasil**.
- **Status Fase 2:** Selesai di sisi pengembangan lokal.
- **Akan Dikerjakan (Next Up):**
  - **Fase 3 — Web Push Notification (VAPID)**: Pembuatan pasangan kunci VAPID, modul pengiriman push, dan tombol aktivasi izin notifikasi di PWA.

---

### 2026-10-12 (lanjutan — Fase 3: Web Push Notification)
- **Fase 3 — Web Push Notification (VAPID): ✅ SELESAI (Sisi Kode)**
  - Menambahkan dependensi `github.com/SherClockHolmes/webpush-go v1.4.0` & `jwt/v5`.
  - Membuat alat CLI generator kunci VAPID di [`cmd/genvapid/main.go`](file:///home/fpans/Projects/mini-assisted/cmd/genvapid/main.go). Kunci dapat langsung digenerate dan dimasukkan ke `.env`.
  - Membangun paket pengiriman push di [`internal/push/service.go`](file:///home/fpans/Projects/mini-assisted/internal/push/service.go):
    - Pengiriman payload notifikasi JSON (`title`, `body`, `data`).
    - Penanganan otomatis pembersihan langganan: jika browser mengembalikan HTTP 404 atau 410 (Gone), data subscription langsung dihapus dari SQLite agar tidak menumpuk.
    - Fungsi `BroadcastAll()` untuk mengirim notifikasi ke semua perangkat pemilik sekaligus.
  - Memperbarui [`internal/server/server.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server.go) dengan endpoint:
    - `GET /api/push/vapid-public-key`: Mengambil public key untuk pendaftaran di browser.
    - `POST /api/push/subscribe`: Menyimpan endpoint dan kunci enkripsi (P256dh & Auth).
    - `POST /api/push/test`: Trigger pengiriman push uji ke semua perangkat pemilik.
    - `GET /settings`: Halaman antarmuka preferensi dan notifikasi.
  - Membuat antarmuka pengaturan dan aktivasi di [`web/templates/settings.html`](file:///home/fpans/Projects/mini-assisted/web/templates/settings.html) dan script klien [`web/static/js/push.js`](file:///home/fpans/Projects/mini-assisted/web/static/js/push.js) untuk meminta izin via `Notification.requestPermission()`.
  - Menambahkan pengujian integrasi endpoint push di [`internal/server/server_test.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server_test.go) (Semua **PASS**).
- **Akan Dikerjakan (Next Up):**
  - **Fase 4 — Reminder Engine & Scheduler**: Background ticker (30–60 detik), pemeriksaan pengingat jatuh tempo, trigger teks persona via Gemini LLM (dengan fallback template netral), dan push otomatis ke HP pemilik.

---

### 2026-10-12 (lanjutan — Fase 4: Reminder Engine & Scheduler)
- **Fase 4 — Reminder Engine & Scheduler: ✅ SELESAI**
  - Membuat paket [`internal/persona/composer.go`](file:///home/fpans/Projects/mini-assisted/internal/persona/composer.go):
    - Meracik pesan pengingat bernuansa personal menggunakan Persona Dara v3 lewat Gemini LLM.
    - Menyesuaikan konteks waktu Asia/Jakarta (pagi, siang, sore, malam).
    - Memiliki fallback template otomatis yang natural jika LLM sedang offline atau kuota habis: `"{waktu} bep, {tugas} udah dikerjakan blm sayangg 💗"`.
  - Membuat background scheduler di [`internal/scheduler/scheduler.go`](file:///home/fpans/Projects/mini-assisted/internal/scheduler/scheduler.go):
    - Ticker periodik (setiap 30 detik) untuk memeriksa daftar pengingat yang berstatus `pending` dan sudah jatuh tempo (`ListDueReminders`).
    - Otomatis memicu `ComposeReminderText()` dan menyiarkan Web Push via `BroadcastAll()`.
    - Menangani pengulangan otomatis: jika `repeat_rule` bernilai `daily` atau `weekly`, pengingat lama ditandai `done` dan otomatis dibuatkan jadwal baru 1 hari / 1 minggu ke depan.
  - Memperbarui API di [`internal/server/server.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server.go) (`DELETE /api/reminders/{id}`).
  - Menghubungkan background worker dan graceful shutdown ke [`cmd/app/main.go`](file:///home/fpans/Projects/mini-assisted/cmd/app/main.go).
  - Menulis unit test [`internal/scheduler/scheduler_test.go`](file:///home/fpans/Projects/mini-assisted/internal/scheduler/scheduler_test.go): eksekusi tick, status updating, dan aturan `daily` reschedule — **PASS 100%**.
  - Seluruh pengujian sistem (`go test ./...`) dan build binary berhasil.
- **Akan Dikerjakan (Next Up):**
  - **Fase 5 — Events & Say Hi**: API jadwal/kalender, briefing pagi (ringkasan event & reminder hari ini), kotak input quick-add bertenaga LLM, dan sapaan "say hi" spontan berkala.

---

### 2026-10-12 (lanjutan — Fase 5: Events & Say Hi)
- **Fase 5 — Tahu Jadwal & Say Hi: ✅ SELESAI**
  - Membuat parser natural language di [`internal/quickadd/parser.go`](file:///home/fpans/Projects/mini-assisted/internal/quickadd/parser.go):
    - Menerima teks bahasa bebas (misal: *"besok jam 2 siang rapat proposal"*).
    - Memanggil Gemini LLM dengan temperature rendah (0.1) untuk mengekstrak entitas ke format terstruktur (`type`, `title`, `start_at_utc`, `end_at_utc`, `repeat_rule`).
    - Fallback aman jika model offline.
  - Memperluas [`internal/persona/composer.go`](file:///home/fpans/Projects/mini-assisted/internal/persona/composer.go):
    - `ComposeMorningBriefing()`: Merangkum jumlah tugas dan acara hari ini dalam sapaan hangat Dara.
    - `ComposeSayHi()`: Meracik sapaan santai spontan yang peka waktu tanpa membahas topik sensitif.
  - Memperbarui background worker di [`internal/scheduler/scheduler.go`](file:///home/fpans/Projects/mini-assisted/internal/scheduler/scheduler.go):
    - **Briefing Pagi Otomatis**: Dikirim setiap pagi antara jam 07:00–08:30 WIB sekali per hari.
    - **Say Hi Berkala**: Mematuhi jam tenang (22:00–08:00 WIB tidak ada sapaan), maksimal 2x per hari dengan jeda minimal 4 jam (pukul 12:00 dan 17:00).
  - Menyediakan endpoint backend di [`internal/server/server.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server.go):
    - `GET/POST/DELETE /api/events`
    - `POST /api/quick-add/parse` & `POST /api/quick-add/confirm`
  - Pengujian integrasi di [`internal/server/server_test.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server_test.go): semua **PASS**.
  - Binary `cmd/app` terkompilasi sukses.
- **Akan Dikerjakan (Next Up):**
  - **Fase 6 — Fitur Chat PWA & Konteks Selektif**: Antarmuka chat di PWA, endpoint `POST /api/chat`, pemanggilan LLM dengan context window hemat (Persona + Facts + Rolling Summary + 10 pesan terakhir), dan rolling summary worker.

---

### 2026-10-12 (lanjutan — Fase 6: Chat PWA dengan Konteks Selektif)
- **Fase 6 — Fitur Chat & Konteks Selektif: ✅ SELESAI**
  - Membangun paket [`internal/chat/service.go`](file:///home/fpans/Projects/mini-assisted/internal/chat/service.go):
    - **Arsitektur Konteks Selektif**: Menyusun prompt cerdas yang hanya mengirim Persona Dara v3 + fakta statis dari tabel `facts` + ringkasan bergulir (`chat_summaries`) + **10 pesan terakhir** saja. Menghemat hingga 90% token LLM dibandingkan mengirim seluruh riwayat.
    - **Rolling Summary Worker Otomatis**: Jika pesan sudah menumpuk (> 30 pesan), otomatis merangkum 20 pesan lama via LLM, menyimpannya ke tabel `chat_summaries`, dan menghapus pesan lawas dari SQLite agar ukuran database tetap ringan dan cepat.
  - Membuat antarmuka obrolan mobile-first di [`web/templates/chat.html`](file:///home/fpans/Projects/mini-assisted/web/templates/chat.html) dengan bubble obrolan responsif, status online, dan optimistic UI update.
  - Memperbarui bilah navigasi bawah (*bottom navigation*) di seluruh layar PWA (`Hari Ini`, `Chat`, `Pengaturan`).
  - Menghubungkan endpoint `GET /chat`, `GET /api/chat/messages`, dan `POST /api/chat` di [`internal/server/server.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server.go).
  - Menulis unit test integrasi obrolan di [`internal/server/server_test.go`](file:///home/fpans/Projects/mini-assisted/internal/server/server_test.go) (Semua **PASS**).
  - Seluruh pengujian sistem (`go test ./...`) dan build binary statis `cmd/app` **PASS 100%**.
- **Tahap Pengujian Lokal:**
  - Seluruh kode fitur (Fase 1 hingga Fase 6) selesai dibangun. Siap diuji coba di localhost sebelum nantinya dideploy ke VPS.

---

### 2026-10-12 (lanjutan — Refactoring UI: Sharp Modern Shapes & Anonymity)
- **Refactoring UI & Shapes: ✅ SELESAI**
  - Mengubah bahasa desain antarmuka dari yang sebelumnya banyak rounded/pills menjadi **Modern Sharp Geometric Shapes** (sudut tajam industrial/tech minimalis, `border-radius: 1-2px`).
  - Menyelaraskan kartu input quick-add, kartu daftar reminder, tombol aksi, form input, dan bubble chat agar mengusung tema tech-flat minimalis yang tegas.
  - **Penyamaran Nama/Identitas (Anonymity):**
    - Menghilangkan badge nama persona (`Dara 💗`) di header layar utama `Hari Ini`.
    - Mengganti judul di layar Chat dari nama spesifik menjadi header netral `OBROLAN` dengan status indikator `● AKTIF`.
  - Semua unit test (`go test ./...`) dan build binary statis **PASS 100%**.

---

### 2026-10-12 (lanjutan — Upgrade Reminder: Todo-List Task Manager & Jadwal Fleksibel)
- **Fitur Todo-List Task & Recurrence Manager: ✅ SELESAI**
  - **Store Layer (`internal/store`):**
    - Menambahkan method `UpdateReminder(r Reminder) error` pada interface `Store` dan implementasi SQLite `sqliteStore` untuk mendukung pengeditan tugas yang sudah ada.
    - Menambahkan unit test di `internal/store/sqlite_test.go`.
  - **Logika Penjadwalan & Aturan Berulang (`internal/scheduler`):**
    - Membuat modul pembantu [`internal/scheduler/repeat.go`](file:///home/fpans/Projects/mini-assisted/internal/scheduler/repeat.go) dengan fungsi `CalculateNextDue(currentDue, repeatRule)`.
    - Mendukung variasi aturan pengulangan komprehensif:
      - `daily` (setiap hari).
      - `weekdays` (Senin–Jumat, otomatis melompati akhir pekan).
      - `weekly` (seminggu sekali pada hari yang sama).
      - `days:1,3,5` (hari tertentu dalam seminggu, misal Senin, Rabu, Jumat).
      - `interval:N` (setiap N hari sekali).
    - Memperbarui `processReminder` agar otomatis menjadwalkan ulang saat reminder jatuh tempo.
    - Menambahkan unit test table-driven lengkap di `internal/scheduler/repeat_test.go` (PASS).
  - **HTTP Endpoints (`internal/server`):**
    - `POST /api/reminders`: Mendukung payload kustom tanggal & jam (`due_at`) dan aturan berulang (`repeat_rule`).
    - `PUT /api/reminders/{id}`: Endpoint untuk mengedit judul tugas, waktu jatuh tempo, atau aturan berulang.
    - `POST /api/reminders/{id}/done`: Menandai selesai dan otomatis memicu pembuatan task berikutnya jika task berulang.
    - `POST /api/reminders/{id}/undone`: Membatalkan tanda selesai (mengembalikan status ke `pending`).
    - Menambahkan unit test integrasi `TestRemindersTaskFlow` di `internal/server/server_test.go` (PASS).
  - **Quick-Add Natural Language (`internal/quickadd`):**
    - Memperluas prompt LLM pada `internal/quickadd/parser.go` agar mampu mendeteksi perintah pengulangan seperti hari kerja (`weekdays`), seminggu berapa kali (`days:1,3,5`), dan interval.
  - **Antarmuka PWA (`web/templates/index.html`, `web/static/js/app.js`, `web/static/css/style.css`):**
    - Menyediakan tombol roda gigi `⚙` di samping tombol `+` quick-add untuk membuka modal detail.
    - Modal interaktif untuk membuat/mengedit tugas: teks tugas, picker `datetime-local`, pilihan pola pengulangan (dengan checkbox hari Sen–Min atau input interval N hari).
    - Tab penyaring status: **Aktif (Pending)** vs **Selesai (Done)**.
    - Badge informatif waktu jatuh tempo (misal *"Hari ini 16:30"*, *"Besok 09:00"*) dan badge jadwal berulang (misal *"🔁 Sen, Rab, Jum"* atau *"🔁 Sen-Jum"*).
    - Interaksi klik tugas untuk edit, centang `✓` untuk selesai/batal, dan tombol `✕` untuk hapus tugas.
  - Seluruh rangkaian pengujian sistem (`go test ./...`) dan build binary mandiri `cmd/app` **PASS 100%**.







