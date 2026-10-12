package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "session"
	sessionDuration   = 30 * 24 * time.Hour // 30 hari

	// Rate limit login: maks 5 percobaan per 15 menit per IP.
	loginMaxAttempts = 5
	loginWindow      = 15 * time.Minute
)

// ---- authManager ----------------------------------------------------------

// authManager mengelola sesi aktif dan rate limiting login.
// Karena hanya satu pengguna, penyimpanan di memori sudah cukup.
type authManager struct {
	passwordHash string

	mu       sync.Mutex
	sessions map[string]time.Time   // token → expiry
	attempts map[string][]time.Time // IP  → waktu percobaan
}

func newAuthManager(passwordHash string) *authManager {
	am := &authManager{
		passwordHash: passwordHash,
		sessions:     make(map[string]time.Time),
		attempts:     make(map[string][]time.Time),
	}
	go am.cleanupLoop()
	return am
}

// cleanupLoop membersihkan sesi dan percobaan yang sudah kedaluwarsa setiap jam.
func (am *authManager) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		am.cleanup()
	}
}

func (am *authManager) cleanup() {
	now := time.Now()
	am.mu.Lock()
	defer am.mu.Unlock()
	for token, exp := range am.sessions {
		if now.After(exp) {
			delete(am.sessions, token)
		}
	}
	for ip, times := range am.attempts {
		var fresh []time.Time
		for _, t := range times {
			if now.Sub(t) < loginWindow {
				fresh = append(fresh, t)
			}
		}
		if len(fresh) == 0 {
			delete(am.attempts, ip)
		} else {
			am.attempts[ip] = fresh
		}
	}
}

// isRateLimited memeriksa apakah IP sudah melebihi batas percobaan login.
func (am *authManager) isRateLimited(ip string) bool {
	now := time.Now()
	am.mu.Lock()
	defer am.mu.Unlock()
	var fresh []time.Time
	for _, t := range am.attempts[ip] {
		if now.Sub(t) < loginWindow {
			fresh = append(fresh, t)
		}
	}
	am.attempts[ip] = fresh
	return len(fresh) >= loginMaxAttempts
}

// recordAttempt mencatat satu percobaan login gagal dari IP.
func (am *authManager) recordAttempt(ip string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.attempts[ip] = append(am.attempts[ip], time.Now())
}

// verifyPassword memverifikasi password terhadap hash bcrypt yang tersimpan.
func (am *authManager) verifyPassword(password string) bool {
	if am.passwordHash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(am.passwordHash), []byte(password)) == nil
}

// newSession membuat token sesi baru dan menyimpannya di memori.
func (am *authManager) newSession() string {
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck — rand.Read tidak pernah error pada implementasi standar
	token := hex.EncodeToString(b)
	am.mu.Lock()
	am.sessions[token] = time.Now().Add(sessionDuration)
	am.mu.Unlock()
	return token
}

// validSession memeriksa apakah token sesi masih valid.
func (am *authManager) validSession(token string) bool {
	if token == "" {
		return false
	}
	am.mu.Lock()
	exp, ok := am.sessions[token]
	am.mu.Unlock()
	return ok && time.Now().Before(exp)
}

// deleteSession menghapus sesi (logout).
func (am *authManager) deleteSession(token string) {
	am.mu.Lock()
	delete(am.sessions, token)
	am.mu.Unlock()
}

// ---- Handler auth ---------------------------------------------------------

// handleLogin memproses login pemilik.
// Menerima JSON {"password":"..."} atau form-data POST biasa.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	if s.auth.isRateLimited(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi nanti")
		return
	}

	var password string
	if r.Header.Get("Content-Type") == "application/json" {
		var body struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "request tidak valid")
			return
		}
		password = body.Password
	} else {
		r.ParseForm()
		password = r.FormValue("password")
	}

	if !s.auth.verifyPassword(password) {
		s.auth.recordAttempt(ip)
		writeError(w, http.StatusUnauthorized, "password salah")
		return
	}

	token := s.auth.newSession()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		// Secure: true dikelola oleh Caddy (HTTPS termination); tidak wajib di lokal
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleLogout menghapus sesi aktif.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.auth.deleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// ---- Middleware -----------------------------------------------------------

// requireAuth memeriksa sesi sebelum meneruskan request ke handler berikutnya.
// Request API (Accept: application/json) mendapat 401 JSON.
// Request browser biasa di-redirect ke /login.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err != nil || !s.auth.validSession(c.Value) {
			if r.Header.Get("Accept") == "application/json" {
				writeError(w, http.StatusUnauthorized, "belum login")
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// ---- Helpers -------------------------------------------------------------

// clientIP mengambil IP asli klien, mempertimbangkan header proxy Caddy.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
