package utils

import (
	"testing"
	"time"
)

func TestIsTradingDay(t *testing.T) {
	// Sabtu → false
	saturday := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC) // Saturday
	if IsTradingDay(saturday) {
		t.Errorf("Saturday should be non-trading day")
	}

	// Minggu → false
	sunday := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)
	if IsTradingDay(sunday) {
		t.Errorf("Sunday should be non-trading day")
	}

	// Hari kerja biasa → true
	wednesday := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	if !IsTradingDay(wednesday) {
		t.Errorf("Wednesday should be a trading day")
	}

	// Hari kerja dengan holiday → false
	// Set a holiday and test
	old := marketHolidays
	marketHolidays = []string{"2026-10-14"}
	defer func() { marketHolidays = old }()

	if IsTradingDay(wednesday) {
		t.Errorf("Wednesday with holiday should be non-trading day")
	}
}
