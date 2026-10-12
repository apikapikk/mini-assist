// Package push menangani enkripsi dan pengiriman Web Push notifikasi via protokol VAPID.
package push

import (
	"encoding/json"
	"fmt"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"mini-assisted/internal/store"
)

// Config konfigurasi kunci VAPID untuk pengiriman push.
type Config struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	SubscriberEmail string // Email kontak pengirim, misal: mailto:owner@domain.com
}

// Payload isi notifikasi yang dikirim ke Service Worker.
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Data  map[string]any `json:"data,omitempty"`
}

// Service mengelola pengiriman notifikasi push ke perangkat klien.
type Service struct {
	cfg   Config
	store store.Store
}

// NewService membuat instance Service baru.
func NewService(cfg Config, st store.Store) *Service {
	if cfg.SubscriberEmail == "" {
		cfg.SubscriberEmail = "mailto:admin@localhost"
	}
	return &Service{
		cfg:   cfg,
		store: st,
	}
}

// Send mengirim push notification ke satu subscription.
// Mengembalikan error jika gagal atau status HTTP bukan 201 Created.
func (s *Service) Send(sub store.PushSubscription, p Payload) error {
	if s.cfg.VAPIDPrivateKey == "" || s.cfg.VAPIDPublicKey == "" {
		return fmt.Errorf("kunci VAPID belum dikonfigurasi")
	}

	payloadBytes, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal payload push: %w", err)
	}

	wpSub := webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	resp, err := webpush.SendNotification(payloadBytes, &wpSub, &webpush.Options{
		Subscriber:      s.cfg.SubscriberEmail,
		VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
		VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
		TTL:             86400, // 24 jam simpan jika perangkat offline
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		return fmt.Errorf("kirim webpush: %w", err)
	}
	defer resp.Body.Close()

	// Jika 404 (Not Found) atau 410 (Gone), artinya langganan sudah kedaluwarsa atau dicabut pengguna di browser
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		_ = s.store.DeletePushSubscription(sub.ID)
		return fmt.Errorf("subscription sudah tidak aktif (status %d), otomatis dihapus", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webpush respon status %d", resp.StatusCode)
	}

	return nil
}

// BroadcastAll mengirim notifikasi ke semua subscription yang tersimpan di DB.
func (s *Service) BroadcastAll(p Payload) (int, error) {
	subs, err := s.store.ListPushSubscriptions()
	if err != nil {
		return 0, fmt.Errorf("ambil daftar push subscription: %w", err)
	}

	successCount := 0
	for _, sub := range subs {
		if err := s.Send(sub, p); err == nil {
			successCount++
		}
	}

	return successCount, nil
}
