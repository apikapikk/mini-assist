package chatlog

import (
	"bufio"
	"io"
	"regexp"
	"strings"
	"time"
)

// Message merepresentasikan satu pesan obrolan yang berhasil diparsing.
type Message struct {
	Time   time.Time `json:"time"`
	Sender string    `json:"sender"`
	Text   string    `json:"text"`
}

// Regex pola baris awal pesan WhatsApp:
// Contoh variasi:
// "12/03/2024 08.15 - Pengirim: teks"
// "12/03/2024, 08:15 - Pengirim: teks"
// "12/03/24, 08.15 - Pengirim: teks"
// "12/03/2024, 8:15 AM - Pengirim: teks"
// "12/03/2024, 08:15:20 - Pengirim: teks"
// "12/03/2024 08.15 - Pesan sistem tanpa titik dua pengirim"
var headerRegex = regexp.MustCompile(`^(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})[,\s]+(\d{1,2}[:.]\d{2}(?::\d{2})?(?:\s*[APap][Mm])?)\s*[-–]\s*(.*)$`)

// Format waktu yang umum digunakan oleh WhatsApp export (Indonesian & English, 12/24 jam)
var timeFormats = []string{
	"02/01/2006 15.04",
	"02/01/2006 15:04",
	"02/01/2006 15.04:05",
	"02/01/2006 15:04:05",
	"02/01/06 15.04",
	"02/01/06 15:04",
	"02/01/06 15.04:05",
	"02/01/06 15:04:05",
	"02/01/2006 3:04 PM",
	"02/01/2006 03:04 PM",
	"02/01/06 3:04 PM",
	"02/01/06 03:04 PM",
	// mm/dd format fallback
	"01/02/2006 15:04",
	"01/02/06 15:04",
}

// Daftar frasa media yang diabaikan
var mediaOmittedPhrases = []string{
	"<media tidak disertakan>",
	"<media omitted>",
	"berkas dilampirkan",
	"file attached",
	"gambar tidak disertakan",
	"stiker tidak disertakan",
	"audio tidak disertakan",
	"video tidak disertakan",
}

// Daftar pesan sistem yang umum
var systemMessagePhrases = []string{
	"pesan dan panggilan dienkripsi",
	"messages and calls are end-to-end encrypted",
	"telah menambahkan",
	"added",
	"keluar dari grup",
	"left",
	"mengubah setelan grup",
	"changed the group settings",
	"mengubah ikon grup",
	"changed this group's icon",
	"mengubah subjek grup",
	"changed the subject",
	"membuat grup",
	"created group",
}

// isSystemMessage mengecek apakah teks adalah notifikasi sistem WhatsApp
func isSystemMessage(sender, text string) bool {
	lowerText := strings.ToLower(text)
	lowerSender := strings.ToLower(sender)

	for _, sys := range systemMessagePhrases {
		if strings.Contains(lowerText, sys) || strings.Contains(lowerSender, sys) {
			return true
		}
	}
	return false
}

// isMediaOmitted mengecek apakah teks hanya berupa penanda media
func isMediaOmitted(text string) bool {
	cleaned := strings.TrimSpace(strings.ToLower(text))
	for _, phrase := range mediaOmittedPhrases {
		if strings.Contains(cleaned, phrase) {
			return true
		}
	}
	return false
}

// parseTimestamp mencoba mencocokkan string tanggal dan jam ke time.Time
func parseTimestamp(dateStr, timeStr string, loc *time.Location) (time.Time, bool) {
	if loc == nil {
		loc = time.Local
	}
	combined := dateStr + " " + strings.ToUpper(strings.TrimSpace(timeStr))

	for _, layout := range timeFormats {
		if t, err := time.ParseInLocation(layout, combined, loc); err == nil {
			return t, true
		}
	}

	// Coba ganti tanda titik dengan titik dua pada jam jika belum cocok
	timeColon := strings.ReplaceAll(timeStr, ".", ":")
	combinedColon := dateStr + " " + strings.ToUpper(strings.TrimSpace(timeColon))
	for _, layout := range timeFormats {
		if t, err := time.ParseInLocation(layout, combinedColon, loc); err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}

// Options untuk parser
type Options struct {
	SenderFilter string         // Jika diset, hanya ambil pesan dari pengirim ini
	Location     *time.Location // Zona waktu parse (default: Asia/Jakarta atau Local)
}

// Parse membaca stream chat WhatsApp dari r dan menghasilkan slice Message yang valid.
func Parse(r io.Reader, opts Options) ([]Message, error) {
	if opts.Location == nil {
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			loc = time.Local
		}
		opts.Location = loc
	}

	scanner := bufio.NewScanner(r)
	var messages []Message
	var currentMsg *Message

	flushCurrent := func() {
		if currentMsg == nil {
			return
		}
		text := strings.TrimSpace(currentMsg.Text)
		if text == "" || isMediaOmitted(text) || isSystemMessage(currentMsg.Sender, text) {
			currentMsg = nil
			return
		}

		// Filter sender jika diset
		if opts.SenderFilter != "" {
			if !strings.EqualFold(strings.TrimSpace(currentMsg.Sender), strings.TrimSpace(opts.SenderFilter)) {
				currentMsg = nil
				return
			}
		}

		currentMsg.Text = text
		messages = append(messages, *currentMsg)
		currentMsg = nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimPrefix(line, "\ufeff") // Hapus UTF-8 BOM jika ada

		matches := headerRegex.FindStringSubmatch(line)
		if matches != nil {
			datePart := matches[1]
			timePart := matches[2]
			rest := matches[3]

			// Cek apakah ada pengirim yang dipisahkan oleh titik dua ':'
			colonIdx := strings.Index(rest, ": ")
			if colonIdx != -1 {
				sender := strings.TrimSpace(rest[:colonIdx])
				text := rest[colonIdx+2:]

				ts, ok := parseTimestamp(datePart, timePart, opts.Location)
				if ok {
					flushCurrent()
					currentMsg = &Message{
						Time:   ts,
						Sender: sender,
						Text:   text,
					}
					continue
				}
			} else {
				// Baris sistem tanpa colon (mis. "12/03/2024 08.15 - Pesan dan panggilan dienkripsi...")
				flushCurrent()
				continue
			}
		}

		// Jika bukan header baru dan ada pesan aktif, ini adalah kelanjutan pesan multi-baris
		if currentMsg != nil {
			currentMsg.Text += "\n" + line
		}
	}

	flushCurrent()

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
