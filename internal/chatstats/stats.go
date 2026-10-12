package chatstats

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"mini-assisted/internal/chatlog"
)

// Options mengatur parameter kalkulasi statistik
type Options struct {
	MinTokenCount int // Ambang batas kemunculan token agar ditampilkan (default: 10)
}

// TokenStat merepresentasikan frekuensi token/frasa
type TokenStat struct {
	Token string `json:"token"`
	Count int    `json:"count"`
}

// Result memuat hasil analisis statistik gaya chat agregat
type Result struct {
	TotalMessages int `json:"total_messages"`
	TotalTurns    int `json:"total_turns"`

	// Panjang pesan
	AvgCharLength    float64 `json:"avg_char_length"`
	MedianCharLength int     `json:"median_char_length"`
	AvgWordCount     float64 `json:"avg_word_count"`
	MedianWordCount  int     `json:"median_word_count"`

	// Bubble per giliran (pesan berturut-turut oleh pengirim yang sama)
	AvgBubblesPerTurn float64 `json:"avg_bubbles_per_turn"`
	MaxBubblesPerTurn int     `json:"max_bubbles_per_turn"`

	// Kapitalisasi & Gaya Huruf
	LowercaseLetterRatio  float64 `json:"lowercase_letter_ratio"`
	UppercaseLetterRatio  float64 `json:"uppercase_letter_ratio"`
	StartsLowercaseRatio  float64 `json:"starts_lowercase_ratio"`  // Proporsi pesan diawali huruf kecil
	AllLowercaseMsgRatio float64 `json:"all_lowercase_msg_ratio"` // Proporsi pesan full huruf kecil

	// Tanda Baca
	NoPunctuationEndRatio float64 `json:"no_punctuation_end_ratio"` // Proporsi pesan tanpa titik/tanda baca di akhir
	ExclamationRatio      float64 `json:"exclamation_msg_ratio"`    // Proporsi pesan mengandung tanda seru
	QuestionRatio         float64 `json:"question_msg_ratio"`       // Proporsi pesan mengandung tanda tanya
	EllipsisRatio         float64 `json:"ellipsis_msg_ratio"`       // Proporsi pesan mengandung ".." atau "..."

	// Pemanjangan Huruf ("iyaaa", "lucuu")
	ElongatedMsgRatio float64     `json:"elongated_msg_ratio"` // Proporsi pesan dengan pemanjangan huruf
	TopElongations    []TokenStat `json:"top_elongations,omitempty"`

	// Emoji
	AvgEmojiPerMsg float64     `json:"avg_emoji_per_msg"`
	TopEmojis      []TokenStat `json:"top_emojis,omitempty"`

	// Pembuka & Penutup
	TopOpeners []TokenStat `json:"top_openers,omitempty"`
	TopClosers []TokenStat `json:"top_closers,omitempty"`

	// Filler & Kata Populer (muncul >= MinTokenCount)
	TopFillersAndWords []TokenStat `json:"top_fillers_and_words,omitempty"`
}

// Analyze menganalisis sekumpulan pesan menjadi statistik agregat
func Analyze(messages []chatlog.Message, opts Options) Result {
	if opts.MinTokenCount <= 0 {
		opts.MinTokenCount = 10
	}

	res := Result{
		TotalMessages: len(messages),
	}
	if len(messages) == 0 {
		return res
	}

	var charLengths []int
	var wordLengths []int
	totalChars := 0
	totalWords := 0

	totalLetters := 0
	totalLowerLetters := 0
	totalUpperLetters := 0

	startsLowerCount := 0
	allLowerCount := 0
	noPunctEndCount := 0
	exclamationCount := 0
	questionCount := 0
	ellipsisCount := 0
	elongatedCount := 0
	totalEmojis := 0

	emojiFreq := make(map[string]int)
	elongationFreq := make(map[string]int)
	openerFreq := make(map[string]int)
	closerFreq := make(map[string]int)
	tokenFreq := make(map[string]int)

	// Hitung giliran (turns)
	turnsCount := 0
	currentTurnBubbles := 0
	var turnBubblesList []int

	for i, m := range messages {
		text := strings.TrimSpace(m.Text)
		charLen := len([]rune(text))
		charLengths = append(charLengths, charLen)
		totalChars += charLen

		words := strings.Fields(text)
		wLen := len(words)
		wordLengths = append(wordLengths, wLen)
		totalWords += wLen

		// Cek giliran pengirim (turn bubble count)
		if i == 0 {
			turnsCount = 1
			currentTurnBubbles = 1
		} else {
			prevSender := messages[i-1].Sender
			// Jika pengirim sama dan selisih waktu wajar
			if m.Sender == prevSender {
				currentTurnBubbles++
			} else {
				turnBubblesList = append(turnBubblesList, currentTurnBubbles)
				turnsCount++
				currentTurnBubbles = 1
			}
		}

		// Huruf besar / kecil
		hasUpper := false
		hasLower := false
		hasLetter := false
		for _, r := range text {
			if unicode.IsLetter(r) {
				hasLetter = true
				totalLetters++
				if unicode.IsLower(r) {
					totalLowerLetters++
					hasLower = true
				} else if unicode.IsUpper(r) {
					totalUpperLetters++
					hasUpper = true
				}
			}
		}

		firstRune := []rune(text)[0]
		if unicode.IsLower(firstRune) {
			startsLowerCount++
		}
		if hasLetter && !hasUpper && hasLower {
			allLowerCount++
		}

		// Tanda baca akhir
		lastRune := []rune(text)[len([]rune(text))-1]
		if !isPunctuation(lastRune) {
			noPunctEndCount++
		}

		if strings.Contains(text, "!") {
			exclamationCount++
		}
		if strings.Contains(text, "?") {
			questionCount++
		}
		if strings.Contains(text, "..") {
			ellipsisCount++
		}

		// Deteksi emoji
		msgEmojiCount := 0
		for _, r := range text {
			if isEmoji(r) {
				emojiStr := string(r)
				emojiFreq[emojiStr]++
				msgEmojiCount++
			}
		}
		totalEmojis += msgEmojiCount

		// Deteksi kata pemanjangan, opener, closer, token
		msgHasElongation := false
		for wi, w := range words {
			cleanWord := cleanToken(w)
			if cleanWord == "" {
				continue
			}

			tokenFreq[cleanWord]++

			if isElongated(cleanWord) {
				msgHasElongation = true
				elongationFreq[cleanWord]++
			}

			// Opener (kata pertama pada pesan)
			if wi == 0 {
				openerFreq[cleanWord]++
			}
			// Closer (kata terakhir pada pesan)
			if wi == len(words)-1 {
				closerFreq[cleanWord]++
			}
		}

		if msgHasElongation {
			elongatedCount++
		}
	}

	if currentTurnBubbles > 0 {
		turnBubblesList = append(turnBubblesList, currentTurnBubbles)
	}

	n := float64(len(messages))

	// Rata-rata & Median
	res.TotalTurns = turnsCount
	res.AvgCharLength = round2(float64(totalChars) / n)
	res.MedianCharLength = medianInt(charLengths)
	res.AvgWordCount = round2(float64(totalWords) / n)
	res.MedianWordCount = medianInt(wordLengths)

	if len(turnBubblesList) > 0 {
		totalBubbles := 0
		maxB := 0
		for _, b := range turnBubblesList {
			totalBubbles += b
			if b > maxB {
				maxB = b
			}
		}
		res.AvgBubblesPerTurn = round2(float64(totalBubbles) / float64(len(turnBubblesList)))
		res.MaxBubblesPerTurn = maxB
	}

	if totalLetters > 0 {
		res.LowercaseLetterRatio = round2(float64(totalLowerLetters) / float64(totalLetters))
		res.UppercaseLetterRatio = round2(float64(totalUpperLetters) / float64(totalLetters))
	}
	res.StartsLowercaseRatio = round2(float64(startsLowerCount) / n)
	res.AllLowercaseMsgRatio = round2(float64(allLowerCount) / n)
	res.NoPunctuationEndRatio = round2(float64(noPunctEndCount) / n)
	res.ExclamationRatio = round2(float64(exclamationCount) / n)
	res.QuestionRatio = round2(float64(questionCount) / n)
	res.EllipsisRatio = round2(float64(ellipsisCount) / n)
	res.ElongatedMsgRatio = round2(float64(elongatedCount) / n)
	res.AvgEmojiPerMsg = round2(float64(totalEmojis) / n)

	// Filter token yang muncul >= opts.MinTokenCount
	res.TopEmojis = filterAndSortTokens(emojiFreq, opts.MinTokenCount, 15)
	res.TopElongations = filterAndSortTokens(elongationFreq, opts.MinTokenCount, 15)
	res.TopOpeners = filterAndSortTokens(openerFreq, opts.MinTokenCount, 15)
	res.TopClosers = filterAndSortTokens(closerFreq, opts.MinTokenCount, 15)
	res.TopFillersAndWords = filterAndSortTokens(tokenFreq, opts.MinTokenCount, 30)

	return res
}

func filterAndSortTokens(m map[string]int, minCount, topN int) []TokenStat {
	var list []TokenStat
	for k, v := range m {
		if v >= minCount {
			list = append(list, TokenStat{Token: k, Count: v})
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count == list[j].Count {
			return list[i].Token < list[j].Token
		}
		return list[i].Count > list[j].Count
	})
	if len(list) > topN {
		list = list[:topN]
	}
	return list
}

func isPunctuation(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == ',' || r == ';' || r == ':' || r == ')' || r == ']' || r == '}'
}

func isEmoji(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1FAFF) || // Miscellaneous Symbols and Pictographs, Emoticons, Supplemental
		(r >= 0x2600 && r <= 0x27BF) || // Dingbats & Misc Symbols
		(r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
		(r >= 0x1F680 && r <= 0x1F6FF) // Transport & Map
}

func isElongated(s string) bool {
	// Minimal 3 karakter berurutan sama, mis: "iyaaa", "lucuuu"
	runes := []rune(s)
	count := 1
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			count++
			if count >= 3 {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

func cleanToken(s string) string {
	s = strings.ToLower(s)
	// Buang tanda baca di awal dan akhir
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r)
	})
}

func medianInt(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	copied := make([]int, len(vals))
	copy(copied, vals)
	sort.Ints(copied)
	mid := len(copied) / 2
	if len(copied)%2 == 0 {
		return (copied[mid-1] + copied[mid]) / 2
	}
	return copied[mid]
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}
