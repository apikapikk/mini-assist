package store_test

import (
	"os"
	"testing"
	"time"

	"mini-assisted/internal/store"
)

// tempDB membuat database SQLite sementara di file temp, dan mengembalikan Store
// beserta fungsi cleanup yang menghapus file setelah test selesai.
func tempDB(t *testing.T) store.Store {
	t.Helper()
	f, err := os.CreateTemp("", "store_test_*.db")
	if err != nil {
		t.Fatalf("buat file temp: %v", err)
	}
	f.Close()

	s, err := store.OpenSQLite(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("buka db: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
		os.Remove(f.Name())
	})
	return s
}

// ---- Reminder -------------------------------------------------------------

func TestReminderCRUD(t *testing.T) {
	s := tempDB(t)

	due := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	// Create
	id, err := s.CreateReminder(store.Reminder{
		Text:     "beli kopi",
		DueAtUTC: due,
	})
	if err != nil {
		t.Fatalf("CreateReminder: %v", err)
	}
	if id == 0 {
		t.Fatal("id harus > 0")
	}

	// Get
	r, err := s.GetReminder(id)
	if err != nil {
		t.Fatalf("GetReminder: %v", err)
	}
	if r.Text != "beli kopi" {
		t.Errorf("text: mau %q, dapat %q", "beli kopi", r.Text)
	}
	if !r.DueAtUTC.Equal(due) {
		t.Errorf("due: mau %v, dapat %v", due, r.DueAtUTC)
	}
	if r.Status != "pending" {
		t.Errorf("status default harus 'pending', dapat %q", r.Status)
	}

	// ListReminders — semua
	list, err := s.ListReminders("")
	if err != nil {
		t.Fatalf("ListReminders: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("panjang list: mau 1, dapat %d", len(list))
	}

	// ListDueReminders
	due2, _ := s.ListDueReminders(due.Add(time.Minute))
	if len(due2) != 1 {
		t.Errorf("ListDueReminders: mau 1, dapat %d", len(due2))
	}
	before, _ := s.ListDueReminders(due.Add(-time.Minute))
	if len(before) != 0 {
		t.Errorf("ListDueReminders sebelum jatuh tempo harus 0, dapat %d", len(before))
	}

	// UpdateReminder
	r.Text = "beli kopi arabika"
	if err := s.UpdateReminder(r); err != nil {
		t.Fatalf("UpdateReminder: %v", err)
	}
	rUpdated, _ := s.GetReminder(id)
	if rUpdated.Text != "beli kopi arabika" {
		t.Errorf("text setelah update: mau %q, dapat %q", "beli kopi arabika", rUpdated.Text)
	}

	// UpdateStatus
	if err := s.UpdateReminderStatus(id, "done"); err != nil {
		t.Fatalf("UpdateReminderStatus: %v", err)
	}
	r, _ = s.GetReminder(id)
	if r.Status != "done" {
		t.Errorf("status setelah update: mau 'done', dapat %q", r.Status)
	}

	// ListReminders filter status
	pending, _ := s.ListReminders("pending")
	if len(pending) != 0 {
		t.Errorf("list pending setelah done harus 0, dapat %d", len(pending))
	}

	// Delete
	if err := s.DeleteReminder(id); err != nil {
		t.Fatalf("DeleteReminder: %v", err)
	}
	_, err = s.GetReminder(id)
	if err == nil {
		t.Error("GetReminder setelah delete harus error")
	}
}

// ---- Event ----------------------------------------------------------------

func TestEventCRUD(t *testing.T) {
	s := tempDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(2 * time.Hour)
	end := now.Add(3 * time.Hour)

	id, err := s.CreateEvent(store.Event{
		Title:      "Meeting mingguan",
		StartAtUTC: start,
		EndAtUTC:   end,
		Notes:      "di ruang zoom",
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	e, err := s.GetEvent(id)
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if e.Title != "Meeting mingguan" {
		t.Errorf("title: mau %q, dapat %q", "Meeting mingguan", e.Title)
	}

	// ListEventsOn hari yang sama
	list, err := s.ListEventsOn(now)
	if err != nil {
		t.Fatalf("ListEventsOn: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListEventsOn: mau 1, dapat %d", len(list))
	}

	// ListEventsOn hari berbeda
	other, _ := s.ListEventsOn(now.Add(48 * time.Hour))
	if len(other) != 0 {
		t.Errorf("ListEventsOn hari lain harus 0, dapat %d", len(other))
	}

	// Update
	e.Title = "Meeting diubah"
	if err := s.UpdateEvent(e); err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	e2, _ := s.GetEvent(id)
	if e2.Title != "Meeting diubah" {
		t.Errorf("title setelah update: mau %q, dapat %q", "Meeting diubah", e2.Title)
	}

	// Delete
	if err := s.DeleteEvent(id); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
	_, err = s.GetEvent(id)
	if err == nil {
		t.Error("GetEvent setelah delete harus error")
	}
}

// ---- Fact -----------------------------------------------------------------

func TestFactUpsert(t *testing.T) {
	s := tempDB(t)

	if err := s.SetFact("nama", "Bep"); err != nil {
		t.Fatalf("SetFact: %v", err)
	}

	f, err := s.GetFact("nama")
	if err != nil {
		t.Fatalf("GetFact: %v", err)
	}
	if f.Value != "Bep" {
		t.Errorf("value: mau %q, dapat %q", "Bep", f.Value)
	}

	// Upsert — value berubah
	if err := s.SetFact("nama", "Bepbep"); err != nil {
		t.Fatalf("SetFact upsert: %v", err)
	}
	f2, _ := s.GetFact("nama")
	if f2.Value != "Bepbep" {
		t.Errorf("value setelah upsert: mau %q, dapat %q", "Bepbep", f2.Value)
	}

	// ListFacts
	if err := s.SetFact("status", "mahasiswa"); err != nil {
		t.Fatalf("SetFact status: %v", err)
	}
	facts, err := s.ListFacts()
	if err != nil {
		t.Fatalf("ListFacts: %v", err)
	}
	if len(facts) != 2 {
		t.Errorf("ListFacts: mau 2, dapat %d", len(facts))
	}

	// Delete
	if err := s.DeleteFact("nama"); err != nil {
		t.Fatalf("DeleteFact: %v", err)
	}
	_, err = s.GetFact("nama")
	if err == nil {
		t.Error("GetFact setelah delete harus error")
	}
}

// ---- Message & ChatSummary ------------------------------------------------

func TestMessageAndSummary(t *testing.T) {
	s := tempDB(t)

	base := time.Now().UTC().Truncate(time.Second)

	for i, pair := range [][2]string{
		{"user", "hei dara"},
		{"model", "hei bep!"},
		{"user", "gimana kabar?"},
		{"model", "baik kok 💗"},
	} {
		_, err := s.SaveMessage(store.Message{
			Role:      pair[0],
			Content:   pair[1],
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatalf("SaveMessage %d: %v", i, err)
		}
	}

	// ListRecentMessages limit 2 — harus kronologis
	msgs, err := s.ListRecentMessages(2)
	if err != nil {
		t.Fatalf("ListRecentMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("panjang: mau 2, dapat %d", len(msgs))
	}
	if msgs[0].Content != "gimana kabar?" {
		t.Errorf("msgs[0]: mau %q, dapat %q", "gimana kabar?", msgs[0].Content)
	}

	// DeleteMessagesBefore
	cutoff := base.Add(2 * time.Minute)
	if err := s.DeleteMessagesBefore(cutoff); err != nil {
		t.Fatalf("DeleteMessagesBefore: %v", err)
	}
	all, _ := s.ListRecentMessages(100)
	if len(all) != 2 {
		t.Errorf("setelah delete, mau 2 pesan tersisa, dapat %d", len(all))
	}

	// SaveChatSummary & LatestChatSummary
	_, err = s.SaveChatSummary(store.ChatSummary{
		Summary:     "Bep dan Dara ngobrol santai.",
		CoversUntil: cutoff,
	})
	if err != nil {
		t.Fatalf("SaveChatSummary: %v", err)
	}

	cs, ok, err := s.LatestChatSummary()
	if err != nil {
		t.Fatalf("LatestChatSummary: %v", err)
	}
	if !ok {
		t.Fatal("harus ada summary")
	}
	if cs.Summary != "Bep dan Dara ngobrol santai." {
		t.Errorf("summary: mau %q, dapat %q", "Bep dan Dara ngobrol santai.", cs.Summary)
	}
}

// ---- PushSubscription -----------------------------------------------------

func TestPushSubscription(t *testing.T) {
	s := tempDB(t)

	sub := store.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/test/abc123",
		P256dh:   "key-abc",
		Auth:     "auth-xyz",
	}

	id, err := s.SavePushSubscription(sub)
	if err != nil {
		t.Fatalf("SavePushSubscription: %v", err)
	}

	list, err := s.ListPushSubscriptions()
	if err != nil {
		t.Fatalf("ListPushSubscriptions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("panjang: mau 1, dapat %d", len(list))
	}

	// Upsert — endpoint sama, key baru
	sub.P256dh = "key-baru"
	_, err = s.SavePushSubscription(sub)
	if err != nil {
		t.Fatalf("SavePushSubscription upsert: %v", err)
	}
	list2, _ := s.ListPushSubscriptions()
	if len(list2) != 1 {
		t.Errorf("setelah upsert tetap harus 1, dapat %d", len(list2))
	}

	// Delete
	if err := s.DeletePushSubscription(id); err != nil {
		t.Fatalf("DeletePushSubscription: %v", err)
	}
	list3, _ := s.ListPushSubscriptions()
	if len(list3) != 0 {
		t.Errorf("setelah delete harus 0, dapat %d", len(list3))
	}
}
