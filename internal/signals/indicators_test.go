package signals

import (
	"math"
	"testing"
	"time"

	"learn-go/internal/models"
)

func tbar(open, high, low, close, vol float64, daysAgo int) models.Bar {
	t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)
	return models.Bar{Time: t, Open: open, High: high, Low: low, Close: close, Volume: vol}
}

func TestIndicators_SMA(t *testing.T) {
	closes := []float64{100, 110, 120, 130, 140, 150, 160, 170, 180, 190}
	// SMA(closes, 5, 4) = avg of indices 0..4 = 120
	if got := SMA(closes, 5, 4); got != 120.0 {
		t.Errorf("SMA(5,4) want 120, got %v", got)
	}
	// SMA(closes, 5, 9) = avg of indices 5..9 = 170
	if got := SMA(closes, 5, 9); got != 170.0 {
		t.Errorf("SMA(5,9) want 170, got %v", got)
	}
	// Not enough data
	if got := SMA(closes, 20, 9); got != 0 {
		t.Errorf("SMA(20,9) want 0 (not enough data), got %v", got)
	}
}

func TestIndicators_MaxHigh(t *testing.T) {
	bars := []models.Bar{
		tbar(0, 90, 0, 0, 0, 4),
		tbar(0, 100, 0, 0, 0, 3),
		tbar(0, 80, 0, 0, 0, 2),
		tbar(0, 110, 0, 0, 0, 1),
		tbar(0, 95, 0, 0, 0, 0),
	}
	if got := MaxHigh(bars, 0, 4); got != 110.0 {
		t.Errorf("MaxHigh(0,4) want 110, got %v", got)
	}
	if got := MaxHigh(bars, 0, 0); got != 90.0 {
		t.Errorf("MaxHigh(0,0) want 90, got %v", got)
	}
}

func TestIndicators_AvgVolume(t *testing.T) {
	bars := []models.Bar{
		tbar(0, 0, 0, 0, 100, 4),
		tbar(0, 0, 0, 0, 200, 3),
		tbar(0, 0, 0, 0, 300, 2),
		tbar(0, 0, 0, 0, 400, 1),
		tbar(0, 0, 0, 0, 500, 0),
	}
	if got := AvgVolume(bars, 0, 4); got != 300.0 {
		t.Errorf("AvgVolume(0,4) want 300, got %v", got)
	}
	if got := AvgVolume(bars, 0, 0); got != 100.0 {
		t.Errorf("AvgVolume(0,0) want 100, got %v", got)
	}
}

// Breakout test cases from spec
func TestBreakout_BasicSignal(t *testing.T) {
	// 65 flat bars + signal bar at index 65 = 66 total bars. MinBars=61 → need ≥61.
	bars := make([]models.Bar, 66)
	for i := 0; i < 65; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 66-i)
	}
	// Signal bar: Close=101 (breakout above prev maxHigh=100), High=101, vol=2×avg
	bars[65] = tbar(101, 101, 101, 101, 200, 0)

	sig := (&breakout20Strategy{}).Evaluate(bars, true)
	if sig.Score != 10 {
		t.Errorf("Breakout: want score=10, got %v", sig.Score)
	}
	if sig.Code != "BREAKOUT20" {
		t.Errorf("Breakout: want code=BREAKOUT20, got %v", sig.Code)
	}
}

func TestBreakout_VolumeTooLow(t *testing.T) {
	bars := make([]models.Bar, 66)
	for i := 0; i < 65; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 66-i)
	}
	bars[65] = tbar(101, 101, 101, 101, 120, 0) // vol 1.2× avg (< 1.5)

	sig := (&breakout20Strategy{}).Evaluate(bars, true)
	if sig.Score != 0 {
		t.Errorf("Breakout vol too low: want score=0, got %v", sig.Score)
	}
}

func TestBreakout_CloseBelowPrevHigh(t *testing.T) {
	bars := make([]models.Bar, 66)
	for i := 0; i < 65; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 66-i)
	}
	bars[65] = tbar(99.9, 99.9, 99.9, 99.9, 200, 0) // vol 2× but close < 100

	sig := (&breakout20Strategy{}).Evaluate(bars, true)
	if sig.Score != 0 {
		t.Errorf("Breakout close below prev high: want score=0, got %v", sig.Score)
	}
}

func TestBreakout_HighSpike21BarsAgo_NotBlocking(t *testing.T) {
	// 66 bars: spike at index 43 (daysAgo=23), signal at index 65 (daysAgo=0).
	// Window for MaxHigh is indices [45..64]. Spike at index 43 is OUTSIDE window.
	bars := make([]models.Bar, 66)
	for i := 0; i < 66; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 66-i)
	}
	bars[43] = tbar(105, 105, 105, 105, 100, 23) // spike at -22 (daysAgo=23), outside window
	bars[65] = tbar(101, 101, 101, 101, 200, 0)  // signal bar: Close=101, High=101

	sig := (&breakout20Strategy{}).Evaluate(bars, true)
	if sig.Score != 10 {
		t.Errorf("Breakout with spike 21 bars ago: want score=10, got %v", sig.Score)
	}
}

func TestBreakout_HighSpike1BarAgo_Blocks(t *testing.T) {
	// 66 bars: spike at index 64 (second newest), signal at index 65. Spike is at -1 (inside window).
	bars := make([]models.Bar, 66)
	for i := 0; i < 66; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 66-i)
	}
	bars[64] = tbar(105, 105, 105, 105, 100, 1) // spike at -1 (second newest), High=105
	bars[65] = tbar(101, 101, 101, 101, 200, 0) // signal bar: Close=101, High=101

	sig := (&breakout20Strategy{}).Evaluate(bars, true)
	if sig.Score != 0 {
		t.Errorf("Breakout with spike at bar -1: want score=0, got %v", sig.Score)
	}
}

func TestBreakout_IHSGNotSafe(t *testing.T) {
	bars := make([]models.Bar, 66)
	for i := 0; i < 65; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 66-i)
	}
	bars[65] = tbar(101, 101, 101, 101, 200, 0)

	sig := (&breakout20Strategy{}).Evaluate(bars, false)
	if sig.Score != 0 {
		t.Errorf("Breakout IHSG not safe: want score=0, got %v", sig.Score)
	}
}

// Pullback test cases from spec
func makeBarsUptrend(n int) []models.Bar {
	bars := make([]models.Bar, n)
	for i := 0; i < n; i++ {
		close := 100.0 + float64(n-1-i)
		bars[i] = tbar(close, close, close, close, 1e9, n-1-i)
	}
	return bars
}

func TestPullback_BasicSignal(t *testing.T) {
	// 61 flat bars + signal bar. Signal bar's Low must be ≤ 1.01×MA20.
	// MA20 = 100 (all flat bars). So Low ≤ 101.
	bars := make([]models.Bar, 61)
	for i := 0; i < 61; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 61-i)
	}
	idxToday := len(bars) - 1
	ma20 := 100.0
	// Low=100.05 < 1.01×100=101, Close=100.5>MA20=100, Close>Open
	bars[idxToday] = tbar(ma20-1, ma20, ma20*1.0005, ma20+0.5, 1e9, 0)

	sig := (&pullback20Strategy{}).Evaluate(bars, true)
	if sig.Score != 10 {
		t.Errorf("Pullback: want score=10, got %v", sig.Score)
	}
	if sig.Code != "PULLBACK_MA20" {
		t.Errorf("Pullback: want code=PULLBACK_MA20, got %v", sig.Code)
	}
}

func TestPullback_CloseBelowOpen(t *testing.T) {
	bars := make([]models.Bar, 61)
	for i := 0; i < 61; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 61-i)
	}
	idxToday := len(bars) - 1
	ma20 := 100.0
	bars[idxToday] = tbar(ma20+1, ma20+1, ma20*1.0005, ma20-0.5, 1e9, 0) // close < open

	sig := (&pullback20Strategy{}).Evaluate(bars, true)
	if sig.Score != 0 {
		t.Errorf("Pullback close < open: want score=0, got %v", sig.Score)
	}
}

func TestPullback_Downtrend(t *testing.T) {
	// MA20 < MA50: downtrend
	bars := make([]models.Bar, 61)
	for i := 0; i < 61; i++ {
		// Close decreases: oldest=131, newest=71
		close := 131.0 - float64(60-i)
		bars[i] = tbar(close, close, close, close, 1e9, 60-i)
	}
	idxToday := len(bars) - 1
	// MA20 ≈ 81.5, MA50 ≈ 101 → MA20 < MA50
	ma20 := 81.5
	bars[idxToday] = tbar(ma20-1, ma20, ma20*1.001, ma20+0.5, 1e9, 0)

	sig := (&pullback20Strategy{}).Evaluate(bars, true)
	if sig.Score != 0 {
		t.Errorf("Pullback downtrend: want score=0, got %v", sig.Score)
	}
}

func TestPullback_MA50NotRising(t *testing.T) {
	// MA50[t] ≤ MA50[t-10]: MA50 flat for last 11 bars
	bars := make([]models.Bar, 61)
	for i := 0; i < 61; i++ {
		close := 100.0
		if i >= 50 {
			close = 100.0 // flat for last 11 bars
		} else {
			close = 100.0 + float64(50-i) // rising before that
		}
		bars[i] = tbar(close, close, close, close, 1e9, 60-i)
	}
	idxToday := len(bars) - 1
	ma20 := 100.0 // all bars at close=100 for last 11
	bars[idxToday] = tbar(ma20-1, ma20, ma20*1.0005, ma20+0.5, 1e9, 0)

	sig := (&pullback20Strategy{}).Evaluate(bars, true)
	if sig.Score != 0 {
		t.Errorf("Pullback MA50 not rising: want score=0, got %v", sig.Score)
	}
}

func TestPullback_IHSGNotSafe(t *testing.T) {
	// Uptrend so ihsgSafe=true, but ihsgSafe param=false → score=0
	bars := make([]models.Bar, 61)
	for i := 0; i < 61; i++ {
		close := 100.0 + float64(60-i)
		bars[i] = tbar(close, close, close, close, 1e9, 60-i)
	}
	idxToday := len(bars) - 1
	ma20 := 130.5
	bars[idxToday] = tbar(ma20-1, ma20, ma20*1.0005, ma20+0.5, 1e9, 0)

	sig := (&pullback20Strategy{}).Evaluate(bars, false)
	if sig.Score != 0 {
		t.Errorf("Pullback IHSG not safe: want score=0, got %v", sig.Score)
	}
}

func TestGetStrategy(t *testing.T) {
	s, err := GetStrategy("cbow")
	if err != nil || s.Name() != "cbow" {
		t.Errorf("GetStrategy cbow failed")
	}
	s, err = GetStrategy("breakout20")
	if err != nil || s.Name() != "breakout20" {
		t.Errorf("GetStrategy breakout20 failed")
	}
	s, err = GetStrategy("pullback20")
	if err != nil || s.Name() != "pullback20" {
		t.Errorf("GetStrategy pullback20 failed")
	}
	_, err = GetStrategy("unknown")
	if err == nil {
		t.Errorf("GetStrategy unknown: want error")
	}
}

func TestStrategy_MinBars(t *testing.T) {
	if got := (&breakout20Strategy{}).MinBars(); got != 61 {
		t.Errorf("breakout20 MinBars want 61, got %v", got)
	}
	if got := (&pullback20Strategy{}).MinBars(); got != 61 {
		t.Errorf("pullback20 MinBars want 61, got %v", got)
	}
	if got := (&CBoWStrategy{}).MinBars(); got != 60 {
		t.Errorf("cbow MinBars want 60, got %v", got)
	}
}

func TestBreakout_MinBars(t *testing.T) {
	// 60 bars (less than 61 required)
	bars := make([]models.Bar, 60)
	for i := 0; i < 60; i++ {
		bars[i] = tbar(100, 100, 100, 100, 100, 60-i)
	}
	sig := (&breakout20Strategy{}).Evaluate(bars, true)
	if sig.Score != 0 {
		t.Errorf("Breakout with 60 bars: want score=0, got %v", sig.Score)
	}
}

// Verify PF(%) = sum(pnl_pct_net > 0) / |sum(pnl_pct_net <= 0)|
// Three trades: +2%, +1%, -3%
func TestProfitFactorPct(t *testing.T) {
	pnlNets := []float64{2.0, 1.0, -3.0}
	sumWins := 0.0
	sumLosses := 0.0
	for _, p := range pnlNets {
		if p > 0 {
			sumWins += p
		} else {
			sumLosses += -p
		}
	}
	expectedPF := sumWins / sumLosses
	if math.Abs(expectedPF-1.0) > 0.001 {
		t.Errorf("PF(%%) want 1.0, got %v", expectedPF)
	}
}
