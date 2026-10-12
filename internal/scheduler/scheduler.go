// Package scheduler menjalankan background loop untuk mengecek reminder yang jatuh tempo,
// memanggil composer persona, dan mengirim push notification.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"


	"mini-assisted/internal/persona"
	"mini-assisted/internal/push"
	"mini-assisted/internal/store"
)

// Scheduler menjalankan pemeriksaan periodik pengingat, briefing, dan say-hi.
type Scheduler struct {
	store    store.Store
	pushSvc  *push.Service
	composer *persona.Composer
	interval time.Duration

	lastBriefingDate string    // "YYYY-MM-DD"
	lastSayHiTime    time.Time // Waktu say-hi terakhir
	sayHiCountToday  int

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New membuat Scheduler baru dengan interval tick (default 30 detik).
func New(st store.Store, pushSvc *push.Service, comp *persona.Composer, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Scheduler{
		store:    st,
		pushSvc:  pushSvc,
		composer: comp,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start menjalankan background tick loop.
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
	log.Printf("[SCHEDULER] Background scheduler aktif (interval: %v)", s.interval)
}

// Stop menghentikan background loop secara anggun (graceful).
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Printf("[SCHEDULER] Background scheduler dihentikan")
}

func (s *Scheduler) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Jalankan sekali saat startup
	s.checkDueReminders()
	s.checkMorningBriefing()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkDueReminders()
			s.checkMorningBriefing()
			s.checkSayHi()
		}
	}
}

// checkDueReminders memeriksa reminder yang pending dan jatuh tempo sekarang atau sebelumnya.
func (s *Scheduler) checkDueReminders() {
	nowUTC := time.Now().UTC()
	dueList, err := s.store.ListDueReminders(nowUTC)
	if err != nil {
		log.Printf("[SCHEDULER] Error mengambil reminder jatuh tempo: %v", err)
		return
	}

	for _, r := range dueList {
		s.processReminder(r)
	}
}

// checkMorningBriefing mengirim rangkuman agenda hari ini pada pagi hari (antara jam 07:00 - 08:30 WIB).
func (s *Scheduler) checkMorningBriefing() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.Local
	}
	nowLocal := time.Now().In(loc)
	todayStr := nowLocal.Format("2006-01-02")

	// Hanya jalan sekali per hari antara jam 7 sampai 8 pagi
	if s.lastBriefingDate == todayStr {
		return
	}
	if nowLocal.Hour() < 7 || nowLocal.Hour() >= 9 {
		return
	}

	// Ambil event dan reminder hari ini
	events, _ := s.store.ListEventsOn(nowLocal.UTC())
	reminders, _ := s.store.ListReminders("pending")

	summaryText := fmt.Sprintf("%d acara dan %d tugas aktif", len(events), len(reminders))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	notifBody := s.composer.ComposeMorningBriefing(ctx, summaryText)
	if s.pushSvc != nil {
		_, _ = s.pushSvc.BroadcastAll(push.Payload{
			Title: "Dara 💗",
			Body:  notifBody,
			Data: map[string]any{"url": "/"},
		})
	}
	s.lastBriefingDate = todayStr
	log.Printf("[SCHEDULER] Briefing pagi terkirim: %q", notifBody)
}

// checkSayHi menyapa santai pemilik secara berkala (maks 2x per hari, patuhi jam tenang 22:00-08:00 WIB).
func (s *Scheduler) checkSayHi() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.Local
	}
	nowLocal := time.Now().In(loc)

	// Jam tenang: jangan say hi antara jam 22.00 malam hingga 08.00 pagi
	if nowLocal.Hour() >= 22 || nowLocal.Hour() < 8 {
		return
	}

	// Reset hitungan harian jika ganti hari
	todayStr := nowLocal.Format("2006-01-02")
	if s.lastSayHiTime.In(loc).Format("2006-01-02") != todayStr {
		s.sayHiCountToday = 0
	}

	// Batas maks 2x say hi per hari
	if s.sayHiCountToday >= 2 {
		return
	}

	// Jarak minimal antar say hi adalah 4 jam
	if !s.lastSayHiTime.IsZero() && nowLocal.Sub(s.lastSayHiTime) < 4*time.Hour {
		return
	}

	// Say hi hanya jika di jam 12 (siang) atau jam 17 (sore)
	h := nowLocal.Hour()
	if h != 12 && h != 17 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	notifBody := s.composer.ComposeSayHi(ctx)
	if s.pushSvc != nil {
		_, _ = s.pushSvc.BroadcastAll(push.Payload{
			Title: "Dara 💗",
			Body:  notifBody,
			Data: map[string]any{"url": "/"},
		})
	}

	s.lastSayHiTime = nowLocal
	s.sayHiCountToday++
	log.Printf("[SCHEDULER] Say Hi spontan terkirim (%d/2 hari ini): %q", s.sayHiCountToday, notifBody)
}


func (s *Scheduler) processReminder(r store.Reminder) {
	log.Printf("[SCHEDULER] Reminder jatuh tempo ID %d: %q", r.ID, r.Text)

	// Buat teks pengingat persona
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	notifBody := s.composer.ComposeReminderText(ctx, r.Text)

	// Kirim Web Push ke semua perangkat pemilik
	if s.pushSvc != nil {
		sentCount, err := s.pushSvc.BroadcastAll(push.Payload{
			Title: "Dara 💗",
			Body:  notifBody,
			Data: map[string]any{
				"url":        "/",
				"reminderId": r.ID,
			},
		})
		if err != nil {
			log.Printf("[SCHEDULER] Gagal broadcast push reminder ID %d: %v", r.ID, err)
		} else {
			log.Printf("[SCHEDULER] Push terkirim ke %d perangkat: %q", sentCount, notifBody)
		}
	}

	// Tangani pengulangan (repeat_rule) atau tandai selesai
	if nextDue, ok := CalculateNextDue(r.DueAtUTC, r.RepeatRule); ok {
		_, _ = s.store.CreateReminder(store.Reminder{
			Text:       r.Text,
			DueAtUTC:   nextDue,
			RepeatRule: r.RepeatRule,
			Status:     "pending",
		})
		_ = s.store.UpdateReminderStatus(r.ID, "done")
	} else {
		// Jika tidak berulang, tandai sebagai "done" setelah dinotifikasikan
		_ = s.store.UpdateReminderStatus(r.ID, "done")
	}
}
