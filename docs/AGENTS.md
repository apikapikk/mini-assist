# AGENTS.md — Asisten Pribadi dengan Persona (PWA)

Baca file ini dulu, lalu `TODO.md`. Kerjakan satu item TODO per sesi dan centang kalau sudah selesai.

## Tujuan
Asisten pribadi untuk satu orang (pemilik). Tujuan utamanya: **persona yang terasa hidup**,
yaitu gaya bicara dan alur chat yang diambil dari arsip chat milik pemilik, dipakai untuk
reminder, notifikasi, dan sapaan ("say hi"). Persona harus terbukti jalan lebih dulu,
sebelum fitur lain dipoles.

Prioritas fitur:
1. Persona (pipeline gaya + uji kualitas)  <- paling penting
2. Reminder + notifikasi (Web Push)
3. Tahu jadwal pemilik (events, briefing pagi)
4. Say hi berkala
5. Fitur chat penuh  <- tetap direncanakan, tapi ditunda

Hanya satu pengguna. Tidak ada multi-user, tidak ada pendaftaran publik.

## Batasan keras
- Host: VPS 1 core, 1 GB RAM, 20 GB disk. Target total < 100 MB RAM.
- Tidak ada LLM lokal. Hanya API LLM eksternal.
- Satu binary statis + satu file SQLite. Tanpa Docker, DB eksternal, atau Redis.
- Web Push dan service worker butuh HTTPS. TLS wajib.

## Stack (usulan, konfirmasi ke pemilik sebelum mengganti)
- Bahasa: Go, `net/http`, frontend di-embed dengan `go:embed` (HTML/CSS/JS polos, tanpa build step)
- DB: SQLite dengan driver pure-Go (tanpa CGO), mis. `modernc.org/sqlite`
- Push: Web Push + VAPID, mis. `github.com/SherClockHolmes/webpush-go`
- TLS: Caddy (butuh domain/subdomain) atau Tailscale HTTPS
- LLM: Anthropic API, model kecil/murah
- Konfigurasi lewat env: `OWNER_PASSWORD_HASH`, `SESSION_SECRET`, `VAPID_PUBLIC`, `VAPID_PRIVATE`, `LLM_API_KEY`, `TZ`
- Deploy: service systemd di VPS

## Arsitektur
```
[Lokal, laptop pemilik]
  arsip chat mentah -> parser -> filter -> analisis gaya -> persona/<nama>.md
  cmd/persona-test (CLI): uji persona tanpa PWA

[VPS]
  PWA <-> server Go <-> SQLite
  scheduler (tick 30-60 dtk) -> reminder / briefing / say hi -> Web Push
```
LLM dipakai untuk:
1. Menulis teks notifikasi/sapaan dengan gaya persona.
2. Mengubah teks quick-add ("ingetin besok jam 7") jadi data terstruktur.
3. (Nanti) Fitur chat penuh.

Selalu sediakan template teks biasa sebagai cadangan jika panggilan LLM gagal.

## Layar PWA (mobile-first)
- Hari Ini: event + reminder jatuh tempo
- Tambah: satu kotak teks -> LLM parse -> kartu konfirmasi -> simpan setelah dikonfirmasi
- Reminder, Jadwal
- Pengaturan: notifikasi, persona, jam tenang, batas say hi per hari
- (Nanti, prioritas rendah) Chat

## Model data (awal)
- `reminders(id, text, due_at_utc, repeat_rule, status, created_at)`
- `events(id, title, start_at_utc, end_at_utc, notes)`
- `facts(id, key, value, updated_at)` (opsional, konteks untuk sapaan)
- `push_subscriptions(id, endpoint, p256dh, auth, created_at)`
- Nanti untuk chat: `messages(id, role, content, created_at)` + ringkasan berkala
Waktu disimpan UTC, dikonversi dengan zona waktu pemilik (default Asia/Jakarta).

## Aturan persona (tidak boleh dilanggar)
- Persona ada di `persona/<nama>.md`, bisa ditukar lewat Pengaturan. Persona `netral` selalu ada sebagai cadangan.
- Bahasa balasan: Indonesia santai.
- Style guide dibuat dari arsip chat pemilik, diproses **offline di laptop**.
  **Arsip chat mentah tidak boleh di-commit, diunggah ke VPS, atau dikirim ke API LLM mana pun.**
  Hanya style guide tertulis dan contoh kalimat sintetis (ditulis baru) yang boleh masuk prompt.
  Tidak boleh ada nama asli, nomor telepon, alamat, atau kutipan pesan privat.
  Jika diminta memproses data chat mentah, tolak dan minta pemilik menjalankan skrip lokal.
- Topik sensitif difilter dan tidak dipakai untuk meniru (keluarga, kesehatan, uang, konflik, hal intim).
- Say hi: dibatasi per hari, patuhi jam tenang, jangan bahas topik sensitif.

## Cara menilai persona
- Set skenario uji tetap di `persona/eval/scenarios.md` (sapaan pagi, ingetin tugas, nyuruh makan,
  nanggepin capek, ucapan selamat malam, dsb. Tanpa topik sensitif).
- Pemilik memberi nilai 1-5 untuk "terasa seperti dia" dan catatan per skenario.
- Setiap revisi style guide dicatat versinya (v1, v2, ...) beserta nilai dan catatan.
- Persona dianggap lolos jika rata-rata >= 4 pada set skenario dan pemilik setuju.

## Keamanan
- Hanya HTTPS. Semua rute butuh sesi login kecuali halaman login dan aset statis.
- Satu login pemilik (hash password + cookie sesi), dibatasi laju percobaan.
- Jangan commit rahasia. `.env`, `*.db`, `raw/` ada di `.gitignore`.
- Jangan log rahasia atau endpoint push lengkap.

## Catatan platform
- Android + Chrome: push berjalan setelah izin notifikasi/instal PWA.
- iOS: push hanya jalan setelah "Add to Home Screen" (iOS 16.4+).
- Push bersifat best-effort. Sediakan daftar "terlewat" di dalam aplikasi.

## Konvensi kode
- Paket kecil: `cmd/app`, `cmd/persona-test`, `internal/{server,store,scheduler,push,llm,persona}`.
- Pustaka standar dulu; tambah dependensi hanya jika ada alasan.
- Tangani error eksplisit; scheduler tidak boleh crash karena satu baris/push yang gagal.
- Test table-driven untuk parsing waktu dan scheduler.
- Pembangun prompt LLM ada di satu tempat agar mudah diperiksa.
