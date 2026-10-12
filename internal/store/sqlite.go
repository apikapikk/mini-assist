// Package store — implementasi SQLite menggunakan modernc.org/sqlite (pure-Go, tanpa CGO).
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// sqliteStore adalah implementasi Store berbasis SQLite.
type sqliteStore struct {
	db *sql.DB
}

// OpenSQLite membuka (atau membuat) database SQLite di path yang diberikan,
// lalu menjalankan migrasi skema secara otomatis.
func OpenSQLite(path string) (Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: buka db: %w", err)
	}

	// Satu koneksi cukup untuk single-user; cegah "database is locked"
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("store: ping db: %w", err)
	}

	s := &sqliteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("store: migrasi: %w", err)
	}
	return s, nil
}

// migrate membuat semua tabel jika belum ada. Aman dijalankan berulang kali.
func (s *sqliteStore) migrate() error {
	_, err := s.db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA foreign_keys=ON;

		CREATE TABLE IF NOT EXISTS reminders (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			text        TEXT    NOT NULL,
			due_at_utc  DATETIME NOT NULL,
			repeat_rule TEXT    NOT NULL DEFAULT '',
			status      TEXT    NOT NULL DEFAULT 'pending',
			created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);

		CREATE TABLE IF NOT EXISTS events (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			title        TEXT    NOT NULL,
			start_at_utc DATETIME NOT NULL,
			end_at_utc   DATETIME NOT NULL,
			notes        TEXT    NOT NULL DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS facts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			key        TEXT    NOT NULL UNIQUE,
			value      TEXT    NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);

		CREATE TABLE IF NOT EXISTS push_subscriptions (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			endpoint   TEXT    NOT NULL UNIQUE,
			p256dh     TEXT    NOT NULL,
			auth       TEXT    NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);

		CREATE TABLE IF NOT EXISTS messages (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			role       TEXT    NOT NULL,
			content    TEXT    NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);

		CREATE TABLE IF NOT EXISTS chat_summaries (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			summary      TEXT    NOT NULL,
			covers_until DATETIME NOT NULL,
			created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);
	`)
	return err
}

// Close menutup koneksi database.
func (s *sqliteStore) Close() error { return s.db.Close() }

// ---- helper ---------------------------------------------------------------

func scanTime(val string) time.Time {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, val); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// ---- Reminder -------------------------------------------------------------

func (s *sqliteStore) CreateReminder(r Reminder) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO reminders (text, due_at_utc, repeat_rule, status) VALUES (?,?,?,?)`,
		r.Text, fmtTime(r.DueAtUTC), r.RepeatRule, coalesce(r.Status, "pending"),
	)
	if err != nil {
		return 0, fmt.Errorf("store: create reminder: %w", err)
	}
	return res.LastInsertId()
}

func (s *sqliteStore) GetReminder(id int64) (Reminder, error) {
	row := s.db.QueryRow(
		`SELECT id, text, due_at_utc, repeat_rule, status, created_at FROM reminders WHERE id=?`, id,
	)
	return scanReminder(row)
}

func (s *sqliteStore) ListReminders(status string) ([]Reminder, error) {
	var rows *sql.Rows
	var err error
	if status == "" {
		rows, err = s.db.Query(`SELECT id, text, due_at_utc, repeat_rule, status, created_at FROM reminders ORDER BY due_at_utc`)
	} else {
		rows, err = s.db.Query(`SELECT id, text, due_at_utc, repeat_rule, status, created_at FROM reminders WHERE status=? ORDER BY due_at_utc`, status)
	}
	if err != nil {
		return nil, fmt.Errorf("store: list reminders: %w", err)
	}
	defer rows.Close()
	return collectReminders(rows)
}

func (s *sqliteStore) ListDueReminders(before time.Time) ([]Reminder, error) {
	rows, err := s.db.Query(
		`SELECT id, text, due_at_utc, repeat_rule, status, created_at FROM reminders WHERE status='pending' AND due_at_utc <= ? ORDER BY due_at_utc`,
		fmtTime(before),
	)
	if err != nil {
		return nil, fmt.Errorf("store: list due reminders: %w", err)
	}
	defer rows.Close()
	return collectReminders(rows)
}

func (s *sqliteStore) UpdateReminderStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE reminders SET status=? WHERE id=?`, status, id)
	return err
}

func (s *sqliteStore) DeleteReminder(id int64) error {
	_, err := s.db.Exec(`DELETE FROM reminders WHERE id=?`, id)
	return err
}

func scanReminder(row *sql.Row) (Reminder, error) {
	var r Reminder
	var due, created string
	err := row.Scan(&r.ID, &r.Text, &due, &r.RepeatRule, &r.Status, &created)
	if err != nil {
		return r, fmt.Errorf("store: scan reminder: %w", err)
	}
	r.DueAtUTC = scanTime(due)
	r.CreatedAt = scanTime(created)
	return r, nil
}

func collectReminders(rows *sql.Rows) ([]Reminder, error) {
	var out []Reminder
	for rows.Next() {
		var r Reminder
		var due, created string
		if err := rows.Scan(&r.ID, &r.Text, &due, &r.RepeatRule, &r.Status, &created); err != nil {
			return nil, err
		}
		r.DueAtUTC = scanTime(due)
		r.CreatedAt = scanTime(created)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- Event ----------------------------------------------------------------

func (s *sqliteStore) CreateEvent(e Event) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO events (title, start_at_utc, end_at_utc, notes) VALUES (?,?,?,?)`,
		e.Title, fmtTime(e.StartAtUTC), fmtTime(e.EndAtUTC), e.Notes,
	)
	if err != nil {
		return 0, fmt.Errorf("store: create event: %w", err)
	}
	return res.LastInsertId()
}

func (s *sqliteStore) GetEvent(id int64) (Event, error) {
	row := s.db.QueryRow(
		`SELECT id, title, start_at_utc, end_at_utc, notes FROM events WHERE id=?`, id,
	)
	return scanEvent(row)
}

func (s *sqliteStore) ListEventsOn(day time.Time) ([]Event, error) {
	// Ambil semua event yang mulai pada hari yang sama (zona UTC).
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	rows, err := s.db.Query(
		`SELECT id, title, start_at_utc, end_at_utc, notes FROM events WHERE start_at_utc >= ? AND start_at_utc < ? ORDER BY start_at_utc`,
		fmtTime(start), fmtTime(end),
	)
	if err != nil {
		return nil, fmt.Errorf("store: list events on: %w", err)
	}
	defer rows.Close()
	return collectEvents(rows)
}

func (s *sqliteStore) UpdateEvent(e Event) error {
	_, err := s.db.Exec(
		`UPDATE events SET title=?, start_at_utc=?, end_at_utc=?, notes=? WHERE id=?`,
		e.Title, fmtTime(e.StartAtUTC), fmtTime(e.EndAtUTC), e.Notes, e.ID,
	)
	return err
}

func (s *sqliteStore) DeleteEvent(id int64) error {
	_, err := s.db.Exec(`DELETE FROM events WHERE id=?`, id)
	return err
}

func scanEvent(row *sql.Row) (Event, error) {
	var e Event
	var start, end string
	err := row.Scan(&e.ID, &e.Title, &start, &end, &e.Notes)
	if err != nil {
		return e, fmt.Errorf("store: scan event: %w", err)
	}
	e.StartAtUTC = scanTime(start)
	e.EndAtUTC = scanTime(end)
	return e, nil
}

func collectEvents(rows *sql.Rows) ([]Event, error) {
	var out []Event
	for rows.Next() {
		var e Event
		var start, end string
		if err := rows.Scan(&e.ID, &e.Title, &start, &end, &e.Notes); err != nil {
			return nil, err
		}
		e.StartAtUTC = scanTime(start)
		e.EndAtUTC = scanTime(end)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---- Fact -----------------------------------------------------------------

func (s *sqliteStore) SetFact(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO facts (key, value, updated_at) VALUES (?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, value, fmtTime(time.Now().UTC()),
	)
	return err
}

func (s *sqliteStore) GetFact(key string) (Fact, error) {
	row := s.db.QueryRow(`SELECT id, key, value, updated_at FROM facts WHERE key=?`, key)
	var f Fact
	var updated string
	err := row.Scan(&f.ID, &f.Key, &f.Value, &updated)
	if err != nil {
		return f, fmt.Errorf("store: get fact %q: %w", key, err)
	}
	f.UpdatedAt = scanTime(updated)
	return f, nil
}

func (s *sqliteStore) ListFacts() ([]Fact, error) {
	rows, err := s.db.Query(`SELECT id, key, value, updated_at FROM facts ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Fact
	for rows.Next() {
		var f Fact
		var updated string
		if err := rows.Scan(&f.ID, &f.Key, &f.Value, &updated); err != nil {
			return nil, err
		}
		f.UpdatedAt = scanTime(updated)
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *sqliteStore) DeleteFact(key string) error {
	_, err := s.db.Exec(`DELETE FROM facts WHERE key=?`, key)
	return err
}

// ---- PushSubscription -----------------------------------------------------

func (s *sqliteStore) SavePushSubscription(sub PushSubscription) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO push_subscriptions (endpoint, p256dh, auth) VALUES (?,?,?) ON CONFLICT(endpoint) DO UPDATE SET p256dh=excluded.p256dh, auth=excluded.auth`,
		sub.Endpoint, sub.P256dh, sub.Auth,
	)
	if err != nil {
		return 0, fmt.Errorf("store: save push sub: %w", err)
	}
	return res.LastInsertId()
}

func (s *sqliteStore) ListPushSubscriptions() ([]PushSubscription, error) {
	rows, err := s.db.Query(`SELECT id, endpoint, p256dh, auth, created_at FROM push_subscriptions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PushSubscription
	for rows.Next() {
		var sub PushSubscription
		var created string
		if err := rows.Scan(&sub.ID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &created); err != nil {
			return nil, err
		}
		sub.CreatedAt = scanTime(created)
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *sqliteStore) DeletePushSubscription(id int64) error {
	_, err := s.db.Exec(`DELETE FROM push_subscriptions WHERE id=?`, id)
	return err
}

// ---- Message --------------------------------------------------------------

func (s *sqliteStore) SaveMessage(m Message) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO messages (role, content, created_at) VALUES (?,?,?)`,
		m.Role, m.Content, fmtTime(coalesceTime(m.CreatedAt)),
	)
	if err != nil {
		return 0, fmt.Errorf("store: save message: %w", err)
	}
	return res.LastInsertId()
}

func (s *sqliteStore) ListRecentMessages(limit int) ([]Message, error) {
	// Ambil N pesan terakhir lalu balik urutan agar kronologis.
	rows, err := s.db.Query(
		`SELECT id, role, content, created_at FROM (SELECT id, role, content, created_at FROM messages ORDER BY id DESC LIMIT ?) ORDER BY id ASC`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var msg Message
		var created string
		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &created); err != nil {
			return nil, err
		}
		msg.CreatedAt = scanTime(created)
		out = append(out, msg)
	}
	return out, rows.Err()
}

func (s *sqliteStore) DeleteMessagesBefore(t time.Time) error {
	_, err := s.db.Exec(`DELETE FROM messages WHERE created_at < ?`, fmtTime(t))
	return err
}

// ---- ChatSummary ----------------------------------------------------------

func (s *sqliteStore) SaveChatSummary(cs ChatSummary) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO chat_summaries (summary, covers_until, created_at) VALUES (?,?,?)`,
		cs.Summary, fmtTime(cs.CoversUntil), fmtTime(coalesceTime(cs.CreatedAt)),
	)
	if err != nil {
		return 0, fmt.Errorf("store: save chat summary: %w", err)
	}
	return res.LastInsertId()
}

func (s *sqliteStore) LatestChatSummary() (ChatSummary, bool, error) {
	row := s.db.QueryRow(
		`SELECT id, summary, covers_until, created_at FROM chat_summaries ORDER BY id DESC LIMIT 1`,
	)
	var cs ChatSummary
	var covers, created string
	err := row.Scan(&cs.ID, &cs.Summary, &covers, &created)
	if err == sql.ErrNoRows {
		return cs, false, nil
	}
	if err != nil {
		return cs, false, fmt.Errorf("store: latest summary: %w", err)
	}
	cs.CoversUntil = scanTime(covers)
	cs.CreatedAt = scanTime(created)
	return cs, true, nil
}

func (s *sqliteStore) DeleteSummariesBefore(t time.Time) error {
	_, err := s.db.Exec(`DELETE FROM chat_summaries WHERE covers_until < ?`, fmtTime(t))
	return err
}

// ---- helper kecil ---------------------------------------------------------

func coalesce(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func coalesceTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}
