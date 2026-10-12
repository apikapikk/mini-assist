// Package store menyediakan interface dan implementasi penyimpanan data berbasis SQLite.
package store

import "time"

// ---- Model ----------------------------------------------------------------

// Reminder adalah satu catatan pengingat milik pemilik.
type Reminder struct {
	ID         int64
	Text       string
	DueAtUTC   time.Time
	RepeatRule string // "" = tidak berulang, "daily", "weekly"
	Status     string // "pending", "done", "missed"
	CreatedAt  time.Time
}

// Event adalah satu entri jadwal/acara.
type Event struct {
	ID        int64
	Title     string
	StartAtUTC time.Time
	EndAtUTC   time.Time
	Notes     string
}

// Fact adalah potongan konteks statis tentang pemilik, dipakai untuk memperkaya
// prompt LLM tanpa mengirim seluruh history.
type Fact struct {
	ID        int64
	Key       string
	Value     string
	UpdatedAt time.Time
}

// PushSubscription adalah subscription Web Push milik satu perangkat pemilik.
type PushSubscription struct {
	ID        int64
	Endpoint  string
	P256dh    string
	Auth      string
	CreatedAt time.Time
}

// Message adalah satu pesan dalam riwayat chat (role: "user" atau "model").
type Message struct {
	ID        int64
	Role      string
	Content   string
	CreatedAt time.Time
}

// ChatSummary adalah ringkasan bergulir (rolling summary) dari percakapan lama,
// dipakai agar LLM tidak perlu membaca semua history dari awal.
type ChatSummary struct {
	ID          int64
	Summary     string
	CoversUntil time.Time // pesan sebelum waktu ini sudah dirangkum
	CreatedAt   time.Time
}

// ---- Interface ------------------------------------------------------------

// Store mendefinisikan semua operasi penyimpanan data yang dibutuhkan aplikasi.
// Setiap metode mengembalikan error eksplisit; tidak ada panic.
type Store interface {
	// Reminder
	CreateReminder(r Reminder) (int64, error)
	GetReminder(id int64) (Reminder, error)
	ListReminders(status string) ([]Reminder, error)        // status="" → semua
	ListDueReminders(before time.Time) ([]Reminder, error)  // jatuh tempo sebelum 'before'
	UpdateReminderStatus(id int64, status string) error
	DeleteReminder(id int64) error

	// Event
	CreateEvent(e Event) (int64, error)
	GetEvent(id int64) (Event, error)
	ListEventsOn(day time.Time) ([]Event, error) // events yang mulai pada hari yang sama
	UpdateEvent(e Event) error
	DeleteEvent(id int64) error

	// Fact
	SetFact(key, value string) error // upsert berdasarkan key
	GetFact(key string) (Fact, error)
	ListFacts() ([]Fact, error)
	DeleteFact(key string) error

	// Push
	SavePushSubscription(sub PushSubscription) (int64, error)
	ListPushSubscriptions() ([]PushSubscription, error)
	DeletePushSubscription(id int64) error

	// Chat / pesan
	SaveMessage(m Message) (int64, error)
	ListRecentMessages(limit int) ([]Message, error)
	DeleteMessagesBefore(t time.Time) error

	// Rolling summary
	SaveChatSummary(s ChatSummary) (int64, error)
	LatestChatSummary() (ChatSummary, bool, error) // bool=ada atau tidak
	DeleteSummariesBefore(t time.Time) error

	// Lifecycle
	Close() error
}
