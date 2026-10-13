package market

import (
	"testing"
	"time"

	"learn-go/internal/models"
)

func TestCleanBars(t *testing.T) {
	now := time.Now()
	raw := []models.Bar{
		{Time: now, Open: 100, High: 105, Low: 99, Close: 102, Volume: 1000},
		{Time: now, Open: 0, High: 105, Low: 99, Close: 102, Volume: 1000},  // Open = 0 → dibuang
		{Time: now, Open: 100, High: 0, Low: 99, Close: 102, Volume: 1000},  // High = 0 → dibuang
		{Time: now, Open: 100, High: 105, Low: 0, Close: 102, Volume: 1000}, // Low = 0 → dibuang
		{Time: now, Open: 100, High: 105, Low: 99, Close: 0, Volume: 1000},  // Close = 0 → dibuang
		{Time: now, Open: 100, High: 105, Low: 99, Close: 102, Volume: 0},   // Volume = 0 → dibuang
		{Time: now, Open: 100, High: 105, Low: 99, Close: 102, Volume: 1000},
	}

	clean := CleanBars(raw)

	if len(clean) != 2 {
		t.Errorf("Expected 2 bars, got %d", len(clean))
	}

	if clean[0].Close != 102 || clean[1].Close != 102 {
		t.Errorf("Unexpected values: %v", clean)
	}
}

func TestCleanBars_AllZeros(t *testing.T) {
	now := time.Now()
	raw := []models.Bar{
		{Time: now, Open: 0, High: 0, Low: 0, Close: 0, Volume: 0},
	}

	clean := CleanBars(raw)

	if len(clean) != 0 {
		t.Errorf("Expected 0 bars, got %d", len(clean))
	}
}
