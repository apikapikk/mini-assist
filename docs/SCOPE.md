# SCOPE — Iterasi 1: Persona jalan dulu

Cakupan: **Fase 0 + Fase 1** di `TODO.md`. Fase 2 ke atas (PWA, push, reminder, chat) TIDAK dikerjakan di iterasi ini.

## Tujuan iterasi
Membuktikan bahwa persona hasil ekstraksi gaya dari arsip chat terasa benar,
diuji lewat CLI di laptop pemilik, sebelum membangun apa pun di sisi server.

## Dalam scope
1. Alat pemrosesan arsip chat (semua Go + pustaka standar, jalan lokal):
   - `cmd/chat-parse`: ekspor chat -> JSONL
   - `cmd/chat-filter`: bersihkan data identitas dan topik sensitif
   - `cmd/chat-stats`: statistik gaya agregat
2. Style guide persona (`persona/<nama>.md`) + contoh kalimat sintetis yang ditulis baru.
3. `cmd/persona-test` (sudah ada kerangkanya) untuk menjalankan skenario terhadap persona.
4. Penilaian kualitas oleh pemilik (`persona/eval/nilai.md`), iterasi v1, v2, dst.

## Di luar scope (jangan dikerjakan)
- Server, PWA, Web Push, HTTPS/Caddy, login, SQLite, scheduler, deploy ke VPS
- Fitur chat, memori fakta, tool calling, say hi, briefing
- Fine-tuning model
- Mengubah stack atau struktur folder tanpa persetujuan pemilik

## Pembagian peran
**Pemilik:**
- Ekspor arsip chat ke luar repo (mis. `~/private/chat/`)
- Menyediakan blocklist nama/tempat (berkas di luar repo)
- Meninjau hasil filter secara manual
- Menjalankan alat di laptop sendiri dan memberi nilai persona

**Agent:**
- Menulis kode, test, dan dokumen
- Hanya memakai `testdata/sample_chat.txt` (sintetis) untuk menguji
- Tidak pernah menyentuh arsip asli (lihat aturan di `AGENTS.md`)

## Tugas agent (urut)
Kerjakan satu per sesi. Setiap tugas selesai jika `go test ./...` hijau dan alatnya jalan di `testdata/`.

**A1. `cmd/chat-parse` + `internal/chatlog`**
- Input: path ekspor WhatsApp `.txt` (asumsi format `dd/mm/yyyy HH.MM - Nama: teks`; periksa variasi: 12/24 jam, pemisah `.` atau `:`, tanggal `d/m/yy`)
- Tangani pesan multi-baris, baris sistem (bergabung/keluar/enkripsi), penanda media ("<Media tidak disertakan>" dan versi bahasa Inggris)
- Flag: `-in`, `-out` (JSONL: `time`, `sender`, `text`), `-sender` (nama yang diambil)
- Test table-driven pada beberapa variasi format sintetis

**A2. `cmd/chat-filter`**
- Hapus/ganti nomor telepon, email, URL, @mention
- `-blocklist <berkas>`: daftar nama/tempat milik pemilik (di luar repo), cocokkan tanpa membedakan huruf besar-kecil
- Daftar kata kunci topik sensitif di `config/sensitif.txt` (generik: keluarga, kesehatan, uang, konflik, intim); pesan yang kena dibuang
- Pesan yang meragukan ditulis ke berkas tinjauan di luar repo untuk dicek pemilik
- Cetak laporan HANYA berupa angka (berapa pesan masuk, dibuang, diragukan)

**A3. `cmd/chat-stats`**
- Keluaran JSON agregat: panjang pesan (rata-rata/median), pesan per giliran, proporsi huruf kecil/kapital, tanda baca, pemanjangan huruf ("iyaaa"), frekuensi emoji, kata pembuka dan penutup percakapan, filler/singkatan
- Token/frasa hanya ditampilkan jika muncul >= 10 kali dan lolos filter; tidak ada kutipan pesan utuh

**A4. Kerangka style guide**
- `persona/_template.md`: bagian sapaan, nada, panjang, filler, emoji, pola membuka/menanggapi/menutup, hal yang dihindari
- Bantu pemilik mengisinya dari statistik A3, dengan contoh kalimat sintetis baru (jangan menyalin pesan asli)

**A5. Penyempurnaan `cmd/persona-test`**
- Flag `-n` (ulang per skenario), keluaran tabel markdown yang siap ditempel ke `persona/eval/nilai.md`
- Perbandingan dua persona berdampingan (`-persona` dan `-persona2`)

## Definisi selesai (iterasi 1)
- [ ] A1-A5 selesai, `go test ./...` hijau
- [ ] Pemilik menjalankan pipeline di laptop dan mendapat style guide v1
- [ ] Persona lulus penilaian: rata-rata >= 4 pada 8 skenario dan pemilik setuju
- [ ] Data mentah dihapus atau diarsipkan offline
- [ ] Hasil dan catatan versi tercatat di `persona/eval/nilai.md`

## Asumsi yang perlu dikonfirmasi pemilik
- Format arsip: ekspor WhatsApp `.txt` (kalau bukan, parser A1 disesuaikan)
- Model LLM: nilai `LLM_MODEL` di `.env` (contoh: `claude-haiku-5-5`)
