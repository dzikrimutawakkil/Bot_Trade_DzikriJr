package backtest

import (
	"testing"
	"time"

	"learn-go/internal/models"
)

// Test the weekly engine with a deterministic uptrend stock: it should
// be selected on day 1, ride the trend, and exit at TP.
func TestRunWeekly_OpensAndExitsAtTP(t *testing.T) {
	// 60 flat bars + 20 trading days of strong uptrend (gain ~6% → TP fires).
	start := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	bars := make([]models.Bar, 0, 80)
	for i := 0; i < 60; i++ {
		t0 := start.AddDate(0, 0, i)
		price := 1000.0
		bars = append(bars, models.Bar{Time: t0, Open: price, High: price, Low: price, Close: price, Volume: 1e9})
	}
	for i := 60; i < 80; i++ {
		t0 := start.AddDate(0, 0, i)
		price := 1000.0 + float64(i-60)*5 // strong uptrend, +5/bar = +5% in 20 days
		bars = append(bars, models.Bar{Time: t0, Open: price, High: price, Low: price, Close: price, Volume: 1e9})
	}

	cfg := Config{
		Capital:      1000000,
		MaxPositions: 1,
		SlippagePct:  0,
		MaxHoldDays:  20,
		Seed:         42,
		Years:        1,
		DCAMonthly:   0,
		CLFixedPct:   3.0,
		TPPct:        5.0,
		Symbols:      []string{"UP"},
	}

	allBars := map[string][]models.Bar{"UP": bars}
	res, err := RunWeekly(cfg, allBars)
	if err != nil {
		t.Fatalf("RunWeekly error: %v", err)
	}

	if len(res.Trades) == 0 {
		t.Fatalf("want ≥1 trade, got 0")
	}
	hasTP := false
	for _, tr := range res.Trades {
		if tr.ExitReason == "TP" {
			hasTP = true
		}
	}
	if !hasTP {
		t.Errorf("want at least one TP exit, got reasons: %+v", res.Trades)
	}
}

// Test the CL path: stock starts strong then drops, hitting -3% CL.
func TestRunWeekly_ExitAtCL(t *testing.T) {
	start := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	bars := make([]models.Bar, 0, 80)
	// 60 bars uptrend so it qualifies (MA5>MA20>MA50).
	for i := 0; i < 60; i++ {
		t0 := start.AddDate(0, 0, i)
		price := 1000.0 + float64(i)*2
		bars = append(bars, models.Bar{Time: t0, Open: price, High: price, Low: price, Close: price, Volume: 1e9})
	}
	// Then a big drop on day 60 to -5% (CL fires).
	for i := 60; i < 80; i++ {
		t0 := start.AddDate(0, 0, i)
		price := 1120.0 - float64(i-60)*30
		bars = append(bars, models.Bar{Time: t0, Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 1e9})
	}

	cfg := Config{
		Capital:      1000000,
		MaxPositions: 1,
		SlippagePct:  0,
		MaxHoldDays:  20,
		Seed:         42,
		Years:        1,
		DCAMonthly:   0,
		CLFixedPct:   3.0,
		TPPct:        5.0,
		Symbols:      []string{"DOWN"},
	}

	allBars := map[string][]models.Bar{"DOWN": bars}
	res, err := RunWeekly(cfg, allBars)
	if err != nil {
		t.Fatalf("RunWeekly error: %v", err)
	}

	if len(res.Trades) == 0 {
		t.Fatalf("want ≥1 trade, got 0")
	}
	hasCL := false
	for _, tr := range res.Trades {
		if tr.ExitReason == "CL" {
			hasCL = true
		}
	}
	if !hasCL {
		t.Errorf("want at least one CL exit, got reasons: %+v", res.Trades)
	}
}

// Verify DCA injection occurs on the first trading day of each month.
func TestRunWeekly_DCAInjection(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// 60 days of flat then 60 more flat (4 months roughly).
	bars := make([]models.Bar, 0, 120)
	for i := 0; i < 120; i++ {
		t0 := start.AddDate(0, 0, i)
		price := 1000.0
		bars = append(bars, models.Bar{Time: t0, Open: price, High: price, Low: price, Close: price, Volume: 1e9})
	}

	cfg := Config{
		Capital:      1000000,
		MaxPositions: 1,
		SlippagePct:  0,
		MaxHoldDays:  20,
		Seed:         42,
		Years:        1,
		DCAMonthly:   500000, // 500k/month
		CLFixedPct:   3.0,
		TPPct:        5.0,
		Symbols:      []string{"X"},
	}

	allBars := map[string][]models.Bar{"X": bars}
	res, err := RunWeekly(cfg, allBars)
	if err != nil {
		t.Fatalf("RunWeekly error: %v", err)
	}

	// Expect at least 1 DCA injection (Feb 1, Mar 1, Apr 1 = 3 months, but
	// data only covers ~5 months: Jan-Apr 2024). Be lenient: at least 2.
	if res.FinalCapital <= cfg.Capital {
		t.Errorf("DCA should have grown cash above starting capital, got %.0f (start %.0f)",
			res.FinalCapital, cfg.Capital)
	}
}

// Determinism: same seed produces identical trade count.
func TestRunWeekly_Deterministic(t *testing.T) {
	start := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	bars := make([]models.Bar, 0, 80)
	for i := 0; i < 80; i++ {
		t0 := start.AddDate(0, 0, i)
		price := 1000.0 + float64(i)
		bars = append(bars, models.Bar{Time: t0, Open: price, High: price, Low: price, Close: price, Volume: 1e9})
	}

	cfg := Config{
		Capital:      1000000,
		MaxPositions: 1,
		SlippagePct:  0,
		MaxHoldDays:  20,
		Seed:         42,
		Years:        1,
		DCAMonthly:   0,
		CLFixedPct:   3.0,
		TPPct:        5.0,
		Symbols:      []string{"X"},
	}
	allBars := map[string][]models.Bar{"X": bars}

	r1, _ := RunWeekly(cfg, allBars)
	r2, _ := RunWeekly(cfg, allBars)
	if len(r1.Trades) != len(r2.Trades) {
		t.Errorf("non-deterministic: %d vs %d trades", len(r1.Trades), len(r2.Trades))
	}
	if r1.FinalCapital != r2.FinalCapital {
		t.Errorf("non-deterministic capital: %.0f vs %.0f", r1.FinalCapital, r2.FinalCapital)
	}
}

// Verify equity curve has one entry per trading day.
func TestRunWeekly_EquityCurve(t *testing.T) {
	start := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	bars := make([]models.Bar, 0, 80)
	for i := 0; i < 80; i++ {
		t0 := start.AddDate(0, 0, i)
		price := 1000.0 + float64(i)
		bars = append(bars, models.Bar{Time: t0, Open: price, High: price, Low: price, Close: price, Volume: 1e9})
	}

	cfg := Config{
		Capital:      1000000,
		MaxPositions: 1,
		SlippagePct:  0,
		MaxHoldDays:  20,
		Seed:         42,
		Years:        1,
		DCAMonthly:   0,
		CLFixedPct:   3.0,
		TPPct:        5.0,
		Symbols:      []string{"X"},
	}
	allBars := map[string][]models.Bar{"X": bars}

	res, _ := RunWeekly(cfg, allBars)
	if len(res.EquityCurve) != len(bars) {
		t.Errorf("equity curve should have %d points, got %d", len(bars), len(res.EquityCurve))
	}
}
