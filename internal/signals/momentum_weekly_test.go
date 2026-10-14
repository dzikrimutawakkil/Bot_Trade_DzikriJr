package signals

import (
	"testing"
	"time"

	"learn-go/internal/models"
)

func makeUptrendBars(price float64, n int) []models.Bar {
	bars := make([]models.Bar, n)
	for i := 0; i < n; i++ {
		p := price + float64(i)*2 // steady uptrend
		t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		bars[i] = models.Bar{Time: t, Open: p, High: p + 5, Low: p - 5, Close: p, Volume: 1e9}
	}
	return bars
}

func makeFlatBars(price float64, n int) []models.Bar {
	bars := make([]models.Bar, n)
	for i := 0; i < n; i++ {
		t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		bars[i] = models.Bar{Time: t, Open: price, High: price, Low: price, Close: price, Volume: 1e9}
	}
	return bars
}

func makeDowntrendBars(price float64, n int) []models.Bar {
	bars := make([]models.Bar, n)
	for i := 0; i < n; i++ {
		p := price - float64(i)*2
		t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		bars[i] = models.Bar{Time: t, Open: p, High: p + 5, Low: p - 5, Close: p, Volume: 1e9}
	}
	return bars
}

func TestSelectWeeklyCandidates_Uptrend(t *testing.T) {
	bars := map[string][]models.Bar{
		"UP1":  makeUptrendBars(1000, 60),
		"UP2":  makeUptrendBars(800, 60),
		"FL":   makeFlatBars(500, 60),
		"DOWN": makeDowntrendBars(2000, 60),
	}
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 59)

	cands := SelectWeeklyCandidates(bars, []string{"UP1", "UP2", "FL", "DOWN"}, day, 5)
	if len(cands) != 2 {
		t.Errorf("want 2 uptrend candidates, got %d", len(cands))
	}
	for _, c := range cands {
		if c.Symbol != "UP1" && c.Symbol != "UP2" {
			t.Errorf("unexpected candidate: %s", c.Symbol)
		}
	}
}

func TestSelectWeeklyCandidates_SortByMomentum(t *testing.T) {
	makeCustomUptrend := func(start, slope float64) []models.Bar {
		bars := make([]models.Bar, 60)
		for i := 0; i < 60; i++ {
			p := start + float64(i)*slope
			t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
			bars[i] = models.Bar{Time: t, Open: p, High: p + 5, Low: p - 5, Close: p, Volume: 1e9}
		}
		return bars
	}
	// "FAST" has steep recent climb → momentum should be higher.
	bars := map[string][]models.Bar{
		"SLOW": makeCustomUptrend(1000, 1),
		"FAST": makeCustomUptrend(1000, 5),
	}
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 59)

	cands := SelectWeeklyCandidates(bars, []string{"SLOW", "FAST"}, day, 5)
	if len(cands) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(cands))
	}
	if cands[0].Symbol != "FAST" {
		t.Errorf("want FAST first (higher momentum), got %s", cands[0].Symbol)
	}
	if cands[0].MomentumPct <= cands[1].MomentumPct {
		t.Errorf("momentum sort broken: %f vs %f", cands[0].MomentumPct, cands[1].MomentumPct)
	}
}

func TestSelectWeeklyCandidates_TopN(t *testing.T) {
	bars := map[string][]models.Bar{
		"A": makeUptrendBars(1000, 60),
		"B": makeUptrendBars(1100, 60),
		"C": makeUptrendBars(1200, 60),
		"D": makeUptrendBars(1300, 60),
	}
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 59)

	cands := SelectWeeklyCandidates(bars, []string{"A", "B", "C", "D"}, day, 2)
	if len(cands) != 2 {
		t.Errorf("want 2 (topN=2), got %d", len(cands))
	}
}

func TestSelectWeeklyCandidates_NotEnoughData(t *testing.T) {
	bars := map[string][]models.Bar{
		"SHORT": makeUptrendBars(1000, 30), // only 30 bars, need 50+
	}
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 29)

	cands := SelectWeeklyCandidates(bars, []string{"SHORT"}, day, 3)
	if len(cands) != 0 {
		t.Errorf("want 0 candidates for short history, got %d", len(cands))
	}
}

func TestTrendBroken(t *testing.T) {
	uptrend := makeUptrendBars(1000, 60)
	day := uptrend[len(uptrend)-1].Time
	if TrendBroken(uptrend, day) {
		t.Error("uptrend should NOT be broken")
	}

	// Reverse to downtrend.
	downtrend := makeDowntrendBars(2000, 60)
	day = downtrend[len(downtrend)-1].Time
	if !TrendBroken(downtrend, day) {
		t.Error("downtrend should be broken")
	}
}
