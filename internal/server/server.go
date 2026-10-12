// Package server menyediakan HTTP server untuk aplikasi mini-assisted.
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"


	"mini-assisted/internal/chat"
	"mini-assisted/internal/push"
	"mini-assisted/internal/quickadd"
	"mini-assisted/internal/store"
	"mini-assisted/web"
)

// Config adalah konfigurasi server yang dibaca dari environment.
type Config struct {
	OwnerPasswordHash string
	SessionSecret     string
	Addr              string
	VAPIDPublicKey    string
	VAPIDPrivateKey   string
	VAPIDSubject      string
	QuickAddParser    *quickadd.Parser
	ChatService       *chat.Service
}

// Server adalah HTTP server utama.
type Server struct {
	store       store.Store
	cfg         Config
	auth        *authManager
	pushService *push.Service
	parser      *quickadd.Parser
	chatService *chat.Service
	mux         *http.ServeMux
}

// New membuat instance Server baru dan mendaftarkan semua rute.
func New(st store.Store, cfg Config) *Server {
	pushSvc := push.NewService(push.Config{
		VAPIDPublicKey:  cfg.VAPIDPublicKey,
		VAPIDPrivateKey: cfg.VAPIDPrivateKey,
		SubscriberEmail: cfg.VAPIDSubject,
	}, st)

	s := &Server{
		store:       st,
		cfg:         cfg,
		auth:        newAuthManager(cfg.OwnerPasswordHash),
		pushService: pushSvc,
		parser:      cfg.QuickAddParser,
		chatService: cfg.ChatService,
		mux:         http.NewServeMux(),
	}


	s.registerRoutes()
	return s
}

// PushService mengembalikan instance push.Service server.
func (s *Server) PushService() *push.Service {
	return s.pushService
}

// ServeHTTP mengimplementasikan http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}


// registerRoutes mendaftarkan semua handler ke mux.
func (s *Server) registerRoutes() {
	// Aset statis & manifest PWA (Publik)
	staticSub, _ := fs.Sub(web.StaticFiles, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	s.mux.HandleFunc("GET /manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		data, _ := web.StaticFiles.ReadFile("static/manifest.json")
		w.Write(data)
	})

	s.mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Service-Worker-Allowed", "/")
		data, _ := web.StaticFiles.ReadFile("static/sw.js")
		w.Write(data)
	})

	// Auth (Publik)
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /login", s.handleLogin)

	// Logout & Health Check
	s.mux.HandleFunc("GET /logout", s.requireAuth(s.handleLogout))
	s.mux.HandleFunc("GET /api/ping", s.requireAuth(s.handlePing))

	// API Web Push
	s.mux.HandleFunc("GET /api/push/vapid-public-key", s.requireAuth(s.handleGetVAPIDKey))
	s.mux.HandleFunc("POST /api/push/subscribe", s.requireAuth(s.handleSubscribePush))
	s.mux.HandleFunc("POST /api/push/test", s.requireAuth(s.handleTestPush))

	// API Reminders
	s.mux.HandleFunc("GET /api/reminders", s.requireAuth(s.handleListReminders))
	s.mux.HandleFunc("POST /api/reminders", s.requireAuth(s.handleCreateReminder))
	s.mux.HandleFunc("POST /api/reminders/{id}/done", s.requireAuth(s.handleMarkDone))
	s.mux.HandleFunc("DELETE /api/reminders/{id}", s.requireAuth(s.handleDeleteReminder))

	// API Events (Jadwal Kegiatan)
	s.mux.HandleFunc("GET /api/events", s.requireAuth(s.handleListEvents))
	s.mux.HandleFunc("POST /api/events", s.requireAuth(s.handleCreateEvent))
	s.mux.HandleFunc("DELETE /api/events/{id}", s.requireAuth(s.handleDeleteEvent))

	// API Quick-Add cerdas
	s.mux.HandleFunc("POST /api/quick-add/parse", s.requireAuth(s.handleQuickAddParse))
	s.mux.HandleFunc("POST /api/quick-add/confirm", s.requireAuth(s.handleQuickAddConfirm))

	// API & Layar Chat
	s.mux.HandleFunc("GET /chat", s.requireAuth(s.handleChatPage))
	s.mux.HandleFunc("GET /api/chat/messages", s.requireAuth(s.handleListMessages))
	s.mux.HandleFunc("POST /api/chat", s.requireAuth(s.handleSendMessage))

	// Halaman Pengaturan (Setting Notifikasi & Persona)
	s.mux.HandleFunc("GET /settings", s.requireAuth(s.handleSettingsPage))

	// Shell PWA Utama (Privat)
	s.mux.HandleFunc("GET /", s.requireAuth(s.handleIndex))
}



// ---- Handlers -------------------------------------------------------------

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && s.auth.validSession(c.Value) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, _ := web.Templates.ReadFile("templates/login.html")
	w.Write(data)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, _ := web.Templates.ReadFile("templates/index.html")
	w.Write(data)
}

func (s *Server) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, _ := web.Templates.ReadFile("templates/settings.html")
	w.Write(data)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- Push Handlers --------------------------------------------------------

func (s *Server) handleGetVAPIDKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"publicKey": s.cfg.VAPIDPublicKey,
	})
}

func (s *Server) handleSubscribePush(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Endpoint == "" {
		writeError(w, http.StatusBadRequest, "data push subscription tidak valid")
		return
	}

	id, err := s.store.SavePushSubscription(store.PushSubscription{
		Endpoint: body.Endpoint,
		P256dh:   body.Keys.P256dh,
		Auth:     body.Keys.Auth,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan subscription")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "subscribed"})
}

func (s *Server) handleTestPush(w http.ResponseWriter, r *http.Request) {
	count, err := s.pushService.BroadcastAll(push.Payload{
		Title: "Mini Assisted 💗",
		Body:  "Tes notifikasi push berhasil masuk ke perangkatmu!",
		Data: map[string]any{
			"url": "/",
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal kirim broadcast push: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "sent",
		"sentTo": count,
	})
}

// ---- Reminders Handlers ---------------------------------------------------

func (s *Server) handleListReminders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	list, err := s.store.ListReminders(status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memuat data reminder")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateReminder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
		writeError(w, http.StatusBadRequest, "teks pengingat wajib diisi")
		return
	}

	id, err := s.store.CreateReminder(store.Reminder{
		Text:     body.Text,
		DueAtUTC: time.Now().UTC().Add(time.Hour),
		Status:   "pending",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan pengingat")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": "created"})
}

func (s *Server) handleMarkDone(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int64
	for _, c := range idStr {
		if c >= '0' && c <= '9' {
			id = id*10 + int64(c-'0')
		}
	}
	if id == 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return
	}

	if err := s.store.UpdateReminderStatus(id, "done"); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memperbarui status pengingat")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "done"})
}

func (s *Server) handleDeleteReminder(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int64
	for _, c := range idStr {
		if c >= '0' && c <= '9' {
			id = id*10 + int64(c-'0')
		}
	}
	if id == 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return
	}

	if err := s.store.DeleteReminder(id); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menghapus pengingat")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---- Events Handlers ------------------------------------------------------

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	dayStr := r.URL.Query().Get("day")
	targetDay := time.Now().UTC()
	if dayStr != "" {
		if t, err := time.Parse("2006-01-02", dayStr); err == nil {
			targetDay = t
		}
	}

	events, err := s.store.ListEventsOn(targetDay)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memuat daftar event: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title      string `json:"title"`
		StartAtUTC string `json:"start_at_utc"`
		EndAtUTC   string `json:"end_at_utc"`
		Notes      string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		writeError(w, http.StatusBadRequest, "judul event wajib diisi")
		return
	}

	start, err := time.Parse(time.RFC3339, body.StartAtUTC)
	if err != nil {
		start = time.Now().UTC().Add(time.Hour)
	}
	end, err := time.Parse(time.RFC3339, body.EndAtUTC)
	if err != nil {
		end = start.Add(time.Hour)
	}

	id, err := s.store.CreateEvent(store.Event{
		Title:      body.Title,
		StartAtUTC: start,
		EndAtUTC:   end,
		Notes:      body.Notes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan event: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": "created"})
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int64
	for _, c := range idStr {
		if c >= '0' && c <= '9' {
			id = id*10 + int64(c-'0')
		}
	}
	if id == 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return
	}

	if err := s.store.DeleteEvent(id); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menghapus event")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---- Quick-Add Handlers ---------------------------------------------------

func (s *Server) handleQuickAddParse(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		writeError(w, http.StatusBadRequest, "teks input wajib diisi")
		return
	}

	if s.parser == nil {
		// Fallback sederhana jika parser belum diset
		writeJSON(w, http.StatusOK, map[string]any{
			"type":         "reminder",
			"title":        body.Text,
			"start_at_utc": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		})
		return
	}

	item, err := s.parser.Parse(r.Context(), body.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mem-parsing teks: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleQuickAddConfirm(w http.ResponseWriter, r *http.Request) {
	var item quickadd.ParsedItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil || item.Title == "" {
		writeError(w, http.StatusBadRequest, "data konfirmasi tidak valid")
		return
	}

	if item.Type == "event" {
		id, err := s.store.CreateEvent(store.Event{
			Title:      item.Title,
			StartAtUTC: item.StartAtUTC,
			EndAtUTC:   item.EndAtUTC,
			Notes:      item.Notes,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "gagal menyimpan event: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "type": "event"})
	} else {
		id, err := s.store.CreateReminder(store.Reminder{
			Text:       item.Title,
			DueAtUTC:   item.StartAtUTC,
			RepeatRule: item.RepeatRule,
			Status:     "pending",
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "gagal menyimpan reminder: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "type": "reminder"})
	}
}

// ---- Chat Handlers --------------------------------------------------------

func (s *Server) handleChatPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, _ := web.Templates.ReadFile("templates/chat.html")
	w.Write(data)
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.store.ListRecentMessages(30)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil pesan: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Message) == "" {
		writeError(w, http.StatusBadRequest, "pesan tidak boleh kosong")
		return
	}

	if s.chatService == nil {
		writeError(w, http.StatusServiceUnavailable, "layanan chat belum diaktifkan")
		return
	}

	reply, err := s.chatService.Reply(r.Context(), strings.TrimSpace(body.Message))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memproses obrolan: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"role":    "model",
		"content": reply,
	})
}




func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
