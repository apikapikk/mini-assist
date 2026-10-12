package chatstats

import (
	"testing"
	"time"

	"mini-assisted/internal/chatlog"
)

func TestAnalyze(t *testing.T) {
	now := time.Now()
	sampleMsgs := []chatlog.Message{
		{Time: now, Sender: "Dia", Text: "pagi! udah sarapan belom?"},
		{Time: now.Add(time.Minute), Sender: "Dia", Text: "iyaaa nanti kabarin yaaa 😊"},
		{Time: now.Add(2 * time.Minute), Sender: "Dia", Text: "wkwk lucu bgt"},
		{Time: now.Add(5 * time.Minute), Sender: "Kamu", Text: "siap! udah kok"},
		{Time: now.Add(6 * time.Minute), Sender: "Dia", Text: "oke deeeh.."},
	}

	opts := Options{
		MinTokenCount: 1, // Untuk testing kecil kita pakai min 1
	}

	res := Analyze(sampleMsgs, opts)

	if res.TotalMessages != 5 {
		t.Errorf("TotalMessages = %d, want 5", res.TotalMessages)
	}

	if res.TotalTurns != 3 {
		t.Errorf("TotalTurns = %d, want 3", res.TotalTurns)
	}

	if res.AvgCharLength <= 0 {
		t.Errorf("AvgCharLength should be > 0, got %f", res.AvgCharLength)
	}

	if res.AvgWordCount <= 0 {
		t.Errorf("AvgWordCount should be > 0, got %f", res.AvgWordCount)
	}

	if res.ElongatedMsgRatio <= 0 {
		t.Errorf("ElongatedMsgRatio should be > 0 (found 'iyaaa', 'yaaa', 'deeeh'), got %f", res.ElongatedMsgRatio)
	}

	if res.AvgEmojiPerMsg <= 0 {
		t.Errorf("AvgEmojiPerMsg should be > 0 (found 😊), got %f", res.AvgEmojiPerMsg)
	}

	// Cek apakah elongation mendeteksi iyaaa
	foundIyaaa := false
	for _, el := range res.TopElongations {
		if el.Token == "iyaaa" {
			foundIyaaa = true
			break
		}
	}
	if !foundIyaaa {
		t.Errorf("expected 'iyaaa' in TopElongations, got %v", res.TopElongations)
	}
}
