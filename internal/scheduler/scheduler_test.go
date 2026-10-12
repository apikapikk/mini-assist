package scheduler_test

import (
	"context"
	"os"
	"testing"
	"time"

	"mini-assisted/internal/persona"
	"mini-assisted/internal/scheduler"
	"mini-assisted/internal/store"
)

func TestSchedulerTickAndRepeat(t *testing.T) {
	// 1. Buat SQLite store temporer
	f, err := os.CreateTemp("", "sched_test_*.db")
	if err != nil {
		t.Fatalf("buat temp db: %v", err)
	}
	f.Close()
	st, err := store.OpenSQLite(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("buka db: %v", err)
	}
	t.Cleanup(func() {
		st.Close()
		os.Remove(f.Name())
	})

	// 2. Setup Persona Composer dengan fallback template (tanpa LLM agar cepat & reliable di unit test)
	composer := persona.NewComposer(nil, "", time.UTC)

	// Pastikan compose fallback berfungsi
	testText := composer.ComposeReminderText(context.Background(), "Belajar Go")
	if testText == "" {
		t.Fatal("hasil compose reminder tidak boleh kosong")
	}

	// 3. Masukkan reminder yang sudah jatuh tempo
	now := time.Now().UTC()
	idOnce, err := st.CreateReminder(store.Reminder{
		Text:       "Minum Air",
		DueAtUTC:   now.Add(-time.Minute), // jatuh tempo 1 menit lalu
		RepeatRule: "",
		Status:     "pending",
	})
	if err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	idDaily, err := st.CreateReminder(store.Reminder{
		Text:       "Olahraga Pagi",
		DueAtUTC:   now.Add(-time.Minute),
		RepeatRule: "daily",
		Status:     "pending",
	})
	if err != nil {
		t.Fatalf("create daily reminder: %v", err)
	}

	// 4. Inisialisasi scheduler dan jalankan sebentar
	sched := scheduler.New(st, nil, composer, 50*time.Millisecond)
	sched.Start()
	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	// 5. Verifikasi reminder yang sekali jalan ditandai 'done'
	rOnce, err := st.GetReminder(idOnce)
	if err != nil {
		t.Fatalf("get reminder once: %v", err)
	}
	if rOnce.Status != "done" {
		t.Errorf("reminder sekali jalan status harus 'done', dapat %q", rOnce.Status)
	}

	// 6. Verifikasi reminder 'daily' ditandai 'done' dan dibuatkan reminder baru untuk besok
	rDaily, err := st.GetReminder(idDaily)
	if err != nil {
		t.Fatalf("get daily reminder: %v", err)
	}
	if rDaily.Status != "done" {
		t.Errorf("reminder daily lama status harus 'done', dapat %q", rDaily.Status)
	}

	// Cek reminder baru yang statusnya pending
	pendingList, err := st.ListReminders("pending")
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pendingList) != 1 {
		t.Fatalf("harus ada 1 reminder daily baru yang pending, dapat %d", len(pendingList))
	}
	if pendingList[0].Text != "Olahraga Pagi" {
		t.Errorf("teks reminder baru: mau 'Olahraga Pagi', dapat %q", pendingList[0].Text)
	}
}
