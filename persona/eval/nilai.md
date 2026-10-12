# Lembar Penilaian Persona (Evaluation Log)

Dokumen ini mencatat evaluasi hasil uji persona berdasarkan 8 skenario pengujian di `persona/eval/scenarios.md`.

Skala penilaian (1 - 5):

- 1: Sangat tidak mirip / kaku seperti chatbot formal
- 2: Kurang pas / masih terasa generik
- 3: Cukup mirip tapi ada beberapa kata yang terasa aneh
- 4: Mirip dan terasa natural
- 5: Sangat hidup dan persis seperti gaya aslinya

Syarat lolos: Rata-rata nilai >= 4.0 dan disetujui oleh pemilik.

---

## Evaluasi v1 (Target: Persona Dara)

Tanggal: 2026-10-11  
Model LLM: Google Gemini

| No  | Skenario                     | Nilai (1-5) | Catatan Pemilik               |
| --- | ---------------------------- | ----------- | ----------------------------- |
| 1   | Sapaan Pagi Hari             | 4           | pembacaan waktu kurang akurat |
| 2   | Mengingatkan Makan Siang     | 3           | pembacaan waktu kurang akurat |
| 3   | Mengingatkan Tugas Penting   | 3           | belum kasih task              |
| 4   | Menanggapi Pemilik Lelah     | 4           | sudah bagus                   |
| 5   | Mengingatkan Tugas Terlewat  | 3           | belum kasih task              |
| 6   | Say Hi Santai Sore Hari      | 3           | kurang akurat waktunya        |
| 7   | Menanggapi Kabar Gembira     | 4           | oke uda bagus                 |
| 8   | Ucapan Malam Menjelang Tidur | 3           | waktu ga akurat               |

**Rata-rata Nilai:** 3.38  
**Keputusan:** [x] Perlu Revisi v2

---

## Evaluasi v2 (Target: Persona Dara v2)

Tanggal: 2026-10-11  
Model LLM: Google Gemini (`gemini-3.1-flash-lite`)  
Fokus Revisi: Kesadaran waktu kontekstual (pagi/siang/sore/malam) dan penyebutan nama tugas secara spesifik.

| No  | Skenario                     | Respon Persona Dara v2 (Live Gemini)                                | Nilai (1-5) | Catatan Pemilik                                                                                                          |
| --- | ---------------------------- | ------------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------ |
| 1   | Sapaan Pagi Hari             | `pagii sayanggg` <br> `udh bangun blm 💗`                           | 4           |                                                                                                                          |
| 2   | Mengingatkan Makan Siang     | `siang bepp` <br> `udh makan siang blm sayangg 💗`                  | 4           |                                                                                                                          |
| 3   | Mengingatkan Tugas Penting   | `siang bep` <br> `15 mnt lg ada jadwal review dokumen nih 💗`       | 2           | kurang natural, jangan kaya yang ini 'bep 15 mnt lg skripsi nih' kuang natural, buat kaya 'skripsinya udah dikerjakann?' |
| 4   | Menanggapi Pemilik Lelah     | `sayangg capek yaa` <br> `istirahat dulu gih bep 💗`                | 4           |                                                                                                                          |
| 5   | Mengingatkan Tugas Terlewat  | `sayanggg` <br> `kirim berkas td udh blm yaa`                       | 4           |                                                                                                                          |
| 6   | Say Hi Santai Sore Hari      | `sore sayangg` <br> `lg apa bep 💗`                                 | 4           |                                                                                                                          |
| 7   | Menanggapi Kabar Gembira     | `ihh hebat bgt sayanggg` <br> `seneng deh proyek akhirnya kelar 💗` | 4           |                                                                                                                          |
| 8   | Ucapan Malam Menjelang Tidur | `tidur gihh sayanggg` <br> `good night 💗`                          | 4           |                                                                                                                          |

**Rata-rata Nilai:** 3.75  
**Keputusan:** [x] Perlu Revisi v3 (Skenario 3 terlalu kaku seperti notif kalender)

---

## Evaluasi v3 (Target: Persona Dara v3)

Tanggal: 2026-10-12  
Model LLM: Google Gemini (`gemini-3.1-flash-lite`)  
Fokus Revisi: Pengingat tugas menjadi natural/percakapan — menanyakan progres seperti pasangan peduli, bukan notif kalender.

| No  | Skenario                     | Respon Persona Dara v3 (Live Gemini)                                                                                                        | Nilai (1-5) | Catatan Pemilik |
| --- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | --------------- |
| 1   | Sapaan Pagi Hari             | -                                                                                                                                           | 4           |                 |
| 2   | Mengingatkan Makan Siang     | -                                                                                                                                           | 4           |                 |
| 3   | Mengingatkan Tugas Penting   | `siang bep` <br> `review dokumennya udah dikerjakann? 💗` / <br> `bentar lg batas waktunya loh` <br> `skripsinya udah beres blm sayangg 💗` | 4           |                 |
| 4   | Menanggapi Pemilik Lelah     | -                                                                                                                                           | 4           |                 |
| 5   | Mengingatkan Tugas Terlewat  | `bep` <br> `laporan mingguannya udah selesai blm 💗`                                                                                        | 4           |                 |
| 6   | Say Hi Santai Sore Hari      | -                                                                                                                                           | 4           |                 |
| 7   | Menanggapi Kabar Gembira     | -                                                                                                                                           | 4           |                 |
| 8   | Ucapan Malam Menjelang Tidur | -                                                                                                                                           | 4           |                 |

**Rata-rata Nilai:** -  
**Keputusan:** [ x ] Lolos / [ -] Perlu Revisi v4
