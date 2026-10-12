package scheduler_test

import (
	"testing"
	"time"

	"mini-assisted/internal/scheduler"
)

func TestCalculateNextDue(t *testing.T) {
	// Buat waktu Senin, 2026-10-12 10:00:00 UTC
	mon := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	if mon.Weekday() != time.Monday {
		t.Fatalf("harus Senin, dapat %v", mon.Weekday())
	}

	tests := []struct {
		name       string
		start      time.Time
		rule       string
		wantOK     bool
		wantTarget time.Time
	}{
		{
			name:       "empty rule",
			start:      mon,
			rule:       "",
			wantOK:     false,
			wantTarget: time.Time{},
		},
		{
			name:       "none rule",
			start:      mon,
			rule:       "none",
			wantOK:     false,
			wantTarget: time.Time{},
		},
		{
			name:       "daily",
			start:      mon,
			rule:       "daily",
			wantOK:     true,
			wantTarget: mon.Add(24 * time.Hour), // Selasa
		},
		{
			name:       "weekly",
			start:      mon,
			rule:       "weekly",
			wantOK:     true,
			wantTarget: mon.Add(7 * 24 * time.Hour), // Senin depan
		},
		{
			name:       "weekdays from Monday",
			start:      mon,
			rule:       "weekdays",
			wantOK:     true,
			wantTarget: mon.Add(24 * time.Hour), // Selasa
		},
		{
			name:       "weekdays from Friday",
			start:      mon.Add(4 * 24 * time.Hour), // Jumat (2026-10-16)
			rule:       "weekdays",
			wantOK:     true,
			wantTarget: mon.Add(7 * 24 * time.Hour), // Lompat ke Senin berikutnya (+3 hari)
		},
		{
			name:       "days:1,3,5 (Mon, Wed, Fri) from Monday",
			start:      mon,
			rule:       "days:1,3,5",
			wantOK:     true,
			wantTarget: mon.Add(2 * 24 * time.Hour), // Rabu (+2 hari)
		},
		{
			name:       "days:1,3,5 from Friday",
			start:      mon.Add(4 * 24 * time.Hour), // Jumat
			rule:       "days:1,3,5",
			wantOK:     true,
			wantTarget: mon.Add(7 * 24 * time.Hour), // Lompat ke Senin (+3 hari)
		},
		{
			name:       "interval:3",
			start:      mon,
			rule:       "interval:3",
			wantOK:     true,
			wantTarget: mon.Add(3 * 24 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := scheduler.CalculateNextDue(tt.start, tt.rule)
			if ok != tt.wantOK {
				t.Fatalf("CalculateNextDue() ok = %v, wantOK = %v", ok, tt.wantOK)
			}
			if ok && !got.Equal(tt.wantTarget) {
				t.Errorf("CalculateNextDue() = %v, want %v", got, tt.wantTarget)
			}
		})
	}
}
