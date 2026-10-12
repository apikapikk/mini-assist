package chatfilter

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"unicode"

	"mini-assisted/internal/chatlog"
)

var (
	phoneRegex = regexp.MustCompile(`(?i)(?:\+62|62|08)[0-9\s\-]{7,15}[0-9]`)
	emailRegex = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	urlRegex   = regexp.MustCompile(`(?i)https?://[^\s]+`)
	mentionRegex = regexp.MustCompile(`@[a-zA-Z0-9._\-]+`)
)

// FilterResult menyimpan hasil klasifikasi dari satu pesan
type FilterResult struct {
	Original     chatlog.Message
	Cleaned      chatlog.Message
	IsDropped    bool
	IsQuestionable bool
	DropReason   string
	DoubtReason  string
}

// Config memuat parameter penyaringan pesan
type Config struct {
	SensitiveKeywords []string
	Blocklist         []string
}

// LoadWordList membaca daftar kata dari berkas, mengabaikan baris kosong dan komentar (#)
func LoadWordList(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		words = append(words, strings.ToLower(line))
	}
	return words, scanner.Err()
}

// Filter menerapkan aturan sanitasi dan deteksi topik sensitif pada satu pesan
func Filter(msg chatlog.Message, cfg Config) FilterResult {
	res := FilterResult{
		Original: msg,
		Cleaned:  msg,
	}

	text := msg.Text
	lowerText := strings.ToLower(text)

	// 1. Cek topik sensitif
	for _, kw := range cfg.SensitiveKeywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if containsWord(lowerText, kw) {
			res.IsDropped = true
			res.DropReason = "Topik sensitif terdeteksi"
			return res
		}
	}

	hasPII := false

	// 2. Sanitasi URL, Email, Phone, Mention
	if urlRegex.MatchString(text) {
		text = urlRegex.ReplaceAllString(text, "[URL]")
		hasPII = true
	}
	if emailRegex.MatchString(text) {
		text = emailRegex.ReplaceAllString(text, "[EMAIL]")
		hasPII = true
	}
	if phoneRegex.MatchString(text) {
		text = phoneRegex.ReplaceAllString(text, "[NOMOR_TELEPON]")
		hasPII = true
	}
	if mentionRegex.MatchString(text) {
		text = mentionRegex.ReplaceAllString(text, "[MENTION]")
		hasPII = true
	}

	// 3. Sanitasi blocklist (nama/tempat)
	hasBlocklistHit := false
	for _, b := range cfg.Blocklist {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		pattern := `(?i)\b` + regexp.QuoteMeta(b) + `\b`
		if re, err := regexp.Compile(pattern); err == nil {
			if re.MatchString(text) {
				text = re.ReplaceAllString(text, "[REDACTED]")
				hasBlocklistHit = true
			}
		}
	}

	res.Cleaned.Text = strings.TrimSpace(text)

	// Jika terdapat PII atau blocklist hit, atau teks setelah disanitasi kosong/hanya tag
	if hasPII || hasBlocklistHit {
		res.IsQuestionable = true
		res.DoubtReason = "Mengandung PII atau entitas blocklist yang disensor"
	}

	return res
}

// containsWord mengecek apakah kata kunci kw ada sebagai kata utuh atau substring (untuk multi-kata)
func containsWord(text, kw string) bool {
	if strings.Contains(kw, " ") {
		return strings.Contains(text, kw)
	}

	idx := strings.Index(text, kw)
	for idx != -1 {
		start := idx
		end := idx + len(kw)

		validStart := (start == 0) || !isWordChar(rune(text[start-1]))
		validEnd := (end == len(text)) || !isWordChar(rune(text[end]))

		if validStart && validEnd {
			return true
		}

		next := strings.Index(text[end:], kw)
		if next == -1 {
			break
		}
		idx = end + next
	}
	return false
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// ProcessStream memproses seluruh stream pesan dan mengembalikan statistik serta slice pesan yang lolos
type Stats struct {
	TotalInput   int `json:"total_input"`
	Accepted     int `json:"accepted"`
	Dropped      int `json:"dropped"`
	Questionable int `json:"questionable"`
}

func ProcessMessages(messages []chatlog.Message, cfg Config) ([]chatlog.Message, []chatlog.Message, Stats) {
	var accepted []chatlog.Message
	var questionable []chatlog.Message
	stats := Stats{TotalInput: len(messages)}

	for _, m := range messages {
		res := Filter(m, cfg)
		if res.IsDropped {
			stats.Dropped++
			continue
		}
		if res.IsQuestionable {
			stats.Questionable++
			questionable = append(questionable, res.Cleaned)
		}
		stats.Accepted++
		accepted = append(accepted, res.Cleaned)
	}

	return accepted, questionable, stats
}
