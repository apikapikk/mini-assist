package scheduler

import (
	"strconv"
	"strings"
	"time"
)

// CalculateNextDue menghitung waktu jatuh tempo berikutnya berdasarkan repeat_rule.
// Jika rule tidak berulang atau tidak valid, mengembalikan (time.Time{}, false).
// Format repeatRule yang didukung:
// - "daily": Setiap hari di jam & menit yang sama.
// - "weekdays": Senin sampai Jumat di jam & menit yang sama.
// - "weekly": Tepat 7 hari setelahnya di jam & menit yang sama.
// - "days:1,3,5": Hari-hari tertentu dalam seminggu (1=Senin, 2=Selasa, ..., 7=Minggu).
// - "interval:N": Setiap N hari sekali (contoh: "interval:2" untuk 2 hari sekali).
func CalculateNextDue(currentDue time.Time, repeatRule string) (time.Time, bool) {
	rule := strings.TrimSpace(repeatRule)
	if rule == "" || rule == "none" {
		return time.Time{}, false
	}

	switch rule {
	case "daily":
		return currentDue.Add(24 * time.Hour), true

	case "weekly":
		return currentDue.Add(7 * 24 * time.Hour), true

	case "weekdays":
		// Cari hari kerja berikutnya (Senin-Jumat)
		next := currentDue.Add(24 * time.Hour)
		for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
			next = next.Add(24 * time.Hour)
		}
		return next, true

	default:
		// Periksa format days:1,3,5
		if strings.HasPrefix(rule, "days:") {
			daysStr := strings.TrimPrefix(rule, "days:")
			parts := strings.Split(daysStr, ",")
			var targetDays []int
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if val, err := strconv.Atoi(p); err == nil && val >= 1 && val <= 7 {
					targetDays = append(targetDays, val)
				}
			}
			if len(targetDays) == 0 {
				return time.Time{}, false
			}

			// Iterasi dari 1 hari ke depan sampai maksimal 7 hari ke depan
			// ISO weekday: Senin=1, Selasa=2, ..., Minggu=7
			for i := 1; i <= 7; i++ {
				cand := currentDue.Add(time.Duration(i) * 24 * time.Hour)
				// cand.Weekday(): Sunday=0, Monday=1, ... Saturday=6
				isoDay := int(cand.Weekday())
				if isoDay == 0 {
					isoDay = 7 // Minggu
				}

				for _, td := range targetDays {
					if isoDay == td {
						return cand, true
					}
				}
			}
			return currentDue.Add(7 * 24 * time.Hour), true
		}

		// Periksa format interval:N
		if strings.HasPrefix(rule, "interval:") {
			nStr := strings.TrimPrefix(rule, "interval:")
			if n, err := strconv.Atoi(strings.TrimSpace(nStr)); err == nil && n > 0 {
				return currentDue.Add(time.Duration(n) * 24 * time.Hour), true
			}
		}

		return time.Time{}, false
	}
}
