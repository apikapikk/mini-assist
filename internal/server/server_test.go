package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"mini-assisted/internal/server"
	"mini-assisted/internal/store"

	"golang.org/x/crypto/bcrypt"
)

// newTestServer membuat server dengan DB sementara dan password "testpass".
func newTestServer(t *testing.T) *server.Server {
	t.Helper()

	// Buat DB sementara
	f, err := os.CreateTemp("", "server_test_*.db")
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

	hash, err := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}

	return server.New(st, server.Config{
		OwnerPasswordHash: string(hash),
		Addr:              ":0",
	})
}

// loginAs melakukan login dan mengembalikan cookie sesi.
func loginAs(t *testing.T, srv *server.Server, password string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"password": password})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	return nil
}

// ---- Test Login -----------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	srv := newTestServer(t)
	c := loginAs(t, srv, "testpass")
	if c == nil {
		t.Fatal("seharusnya ada cookie sesi setelah login berhasil")
	}
	if c.MaxAge <= 0 {
		t.Errorf("MaxAge harus > 0, dapat %d", c.MaxAge)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	srv := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"password": "salah"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("mau 401, dapat %d", rec.Code)
	}
}

func TestLogin_RateLimit(t *testing.T) {
	srv := newTestServer(t)

	// Kirim 5 percobaan gagal
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(map[string]string{"password": "salah"})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		srv.ServeHTTP(httptest.NewRecorder(), req)
	}

	// Percobaan ke-6 harus kena rate limit (429)
	body, _ := json.Marshal(map[string]string{"password": "salah"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("mau 429, dapat %d", rec.Code)
	}
}

// ---- Test Auth Middleware -------------------------------------------------

func TestRequireAuth_WithoutSession(t *testing.T) {
	srv := newTestServer(t)

	// Request API tanpa sesi → 401 JSON
	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("mau 401, dapat %d", rec.Code)
	}
}

func TestRequireAuth_WithValidSession(t *testing.T) {
	srv := newTestServer(t)
	c := loginAs(t, srv, "testpass")
	if c == nil {
		t.Fatal("login gagal")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Accept", "application/json")
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("mau 200, dapat %d", rec.Code)
	}
}

func TestRequireAuth_BrowserRedirect(t *testing.T) {
	srv := newTestServer(t)

	// Request browser tanpa sesi → redirect /login
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Errorf("mau 302, dapat %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("redirect harus ke /login, dapat %q", loc)
	}
}

// ---- Test Logout ----------------------------------------------------------

func TestLogout(t *testing.T) {
	srv := newTestServer(t)
	c := loginAs(t, srv, "testpass")
	if c == nil {
		t.Fatal("login gagal")
	}

	// Logout
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("logout: mau 200, dapat %d", rec.Code)
	}

	// Coba akses dengan sesi yang sudah dihapus → harus 401 / redirect
	req2 := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req2.Header.Set("Accept", "application/json")
	req2.AddCookie(c)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("setelah logout mau 401, dapat %d", rec2.Code)
	}
}

// ---- Test Push Endpoints --------------------------------------------------

func TestPushEndpoints(t *testing.T) {
	srv := newTestServer(t)
	c := loginAs(t, srv, "testpass")
	if c == nil {
		t.Fatal("login gagal")
	}

	// 1. GET /api/push/vapid-public-key
	reqKey := httptest.NewRequest(http.MethodGet, "/api/push/vapid-public-key", nil)
	reqKey.AddCookie(c)
	recKey := httptest.NewRecorder()
	srv.ServeHTTP(recKey, reqKey)
	if recKey.Code != http.StatusOK {
		t.Errorf("GET vapid key: mau 200, dapat %d", recKey.Code)
	}

	// 2. POST /api/push/subscribe
	subPayload := `{"endpoint":"https://fcm.googleapis.com/test","keys":{"p256dh":"key123","auth":"auth123"}}`
	reqSub := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", bytes.NewBufferString(subPayload))
	reqSub.Header.Set("Content-Type", "application/json")
	reqSub.AddCookie(c)
	recSub := httptest.NewRecorder()
	srv.ServeHTTP(recSub, reqSub)
	if recSub.Code != http.StatusOK {
		t.Errorf("POST subscribe: mau 200, dapat %d (%s)", recSub.Code, recSub.Body.String())
	}
}

// ---- Test Reminders Task Flow ---------------------------------------------

func TestRemindersTaskFlow(t *testing.T) {
	srv := newTestServer(t)
	c := loginAs(t, srv, "testpass")
	if c == nil {
		t.Fatal("login gagal")
	}

	// 1. POST /api/reminders dengan custom due_at dan repeat_rule
	payload := `{"text":"Gym Sore","due_at":"2026-10-12T16:00:00Z","repeat_rule":"days:1,3,5"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/reminders", bytes.NewBufferString(payload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.AddCookie(c)
	recCreate := httptest.NewRecorder()
	srv.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("POST /api/reminders: mau 201, dapat %d (%s)", recCreate.Code, recCreate.Body.String())
	}

	var resCreate struct {
		ID int64 `json:"id"`
	}
	json.NewDecoder(recCreate.Body).Decode(&resCreate)
	if resCreate.ID == 0 {
		t.Fatal("ID harus > 0")
	}

	// 2. PUT /api/reminders/{id} (Update task)
	updatePayload := `{"text":"Gym Sore Intensif","repeat_rule":"days:1,3,5"}`
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/reminders/"+bytes.NewBufferString(string(rune('0'+resCreate.ID))).String(), bytes.NewBufferString(updatePayload))
	reqUpdate = httptest.NewRequest(http.MethodPut, "/api/reminders/"+recCreate.Body.String(), bytes.NewBufferString(updatePayload))
	// path value id
	reqUpdate = httptest.NewRequest(http.MethodPut, "/api/reminders/1", bytes.NewBufferString(updatePayload))
	reqUpdate.Header.Set("Content-Type", "application/json")
	reqUpdate.AddCookie(c)
	recUpdate := httptest.NewRecorder()
	srv.ServeHTTP(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusOK {
		t.Fatalf("PUT /api/reminders/1: mau 200, dapat %d (%s)", recUpdate.Code, recUpdate.Body.String())
	}

	// 3. POST /api/reminders/{id}/done (Tandai selesai & picu recurring task berikutnya)
	reqDone := httptest.NewRequest(http.MethodPost, "/api/reminders/1/done", nil)
	reqDone.AddCookie(c)
	recDone := httptest.NewRecorder()
	srv.ServeHTTP(recDone, reqDone)
	if recDone.Code != http.StatusOK {
		t.Fatalf("POST /api/reminders/1/done: mau 200, dapat %d", recDone.Code)
	}

	// 4. GET /api/reminders?status=done
	reqListDone := httptest.NewRequest(http.MethodGet, "/api/reminders?status=done", nil)
	reqListDone.AddCookie(c)
	recListDone := httptest.NewRecorder()
	srv.ServeHTTP(recListDone, reqListDone)
	if recListDone.Code != http.StatusOK {
		t.Fatalf("GET /api/reminders?status=done: mau 200, dapat %d", recListDone.Code)
	}

	// 5. POST /api/reminders/{id}/undone
	reqUndone := httptest.NewRequest(http.MethodPost, "/api/reminders/1/undone", nil)
	reqUndone.AddCookie(c)
	recUndone := httptest.NewRecorder()
	srv.ServeHTTP(recUndone, reqUndone)
	if recUndone.Code != http.StatusOK {
		t.Fatalf("POST /api/reminders/1/undone: mau 200, dapat %d", recUndone.Code)
	}

	// 6. DELETE /api/reminders/{id}
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/reminders/1", nil)
	reqDel.AddCookie(c)
	recDel := httptest.NewRecorder()
	srv.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("DELETE /api/reminders/1: mau 200, dapat %d", recDel.Code)
	}
}

// ---- Test Events & Quick-Add Endpoints ------------------------------------

func TestEventsAndQuickAdd(t *testing.T) {
	srv := newTestServer(t)
	c := loginAs(t, srv, "testpass")
	if c == nil {
		t.Fatal("login gagal")
	}

	// 1. POST /api/events
	eventPayload := `{"title":"Ujian Skripsi","start_at_utc":"2026-10-15T09:00:00Z","end_at_utc":"2026-10-15T11:00:00Z"}`
	reqEv := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(eventPayload))
	reqEv.Header.Set("Content-Type", "application/json")
	reqEv.AddCookie(c)
	recEv := httptest.NewRecorder()
	srv.ServeHTTP(recEv, reqEv)
	if recEv.Code != http.StatusCreated {
		t.Fatalf("POST event: mau 201, dapat %d (%s)", recEv.Code, recEv.Body.String())
	}

	// 2. GET /api/events
	reqGetEv := httptest.NewRequest(http.MethodGet, "/api/events?day=2026-10-15", nil)
	reqGetEv.AddCookie(c)
	recGetEv := httptest.NewRecorder()
	srv.ServeHTTP(recGetEv, reqGetEv)
	if recGetEv.Code != http.StatusOK {
		t.Fatalf("GET events: mau 200, dapat %d", recGetEv.Code)
	}

	// 3. POST /api/quick-add/confirm
	confirmPayload := `{"type":"reminder","title":"Review jurnal","start_at_utc":"2026-10-13T10:00:00Z"}`
	reqQa := httptest.NewRequest(http.MethodPost, "/api/quick-add/confirm", bytes.NewBufferString(confirmPayload))
	reqQa.Header.Set("Content-Type", "application/json")
	reqQa.AddCookie(c)
	recQa := httptest.NewRecorder()
	srv.ServeHTTP(recQa, reqQa)
	if recQa.Code != http.StatusCreated {
		t.Fatalf("POST quick-add confirm: mau 201, dapat %d (%s)", recQa.Code, recQa.Body.String())
	}
}

// ---- Test Chat Endpoints --------------------------------------------------

func TestChatEndpoints(t *testing.T) {
	srv := newTestServer(t)
	c := loginAs(t, srv, "testpass")
	if c == nil {
		t.Fatal("login gagal")
	}

	// 1. GET /chat (halaman chat PWA)
	reqPage := httptest.NewRequest(http.MethodGet, "/chat", nil)
	reqPage.AddCookie(c)
	recPage := httptest.NewRecorder()
	srv.ServeHTTP(recPage, reqPage)
	if recPage.Code != http.StatusOK {
		t.Fatalf("GET /chat: mau 200, dapat %d", recPage.Code)
	}

	// 2. GET /api/chat/messages
	reqList := httptest.NewRequest(http.MethodGet, "/api/chat/messages", nil)
	reqList.AddCookie(c)
	recList := httptest.NewRecorder()
	srv.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("GET /api/chat/messages: mau 200, dapat %d", recList.Code)
	}
}



