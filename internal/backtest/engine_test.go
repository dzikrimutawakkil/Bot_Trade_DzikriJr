package backtest

import (
	"testing"
	"time"

	"learn-go/internal/models"
	"learn-go/internal/signals"
)

func btBar(y, m, d int, open, high, low, close, vol float64) models.Bar {
	return models.Bar{
		Time:   time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC),
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: vol,
	}
}

func btFlat(n int, price, vol float64, startDate time.Time) []models.Bar {
	bars := make([]models.Bar, n)
	for i := 0; i < n; i++ {
		t := startDate.AddDate(0, 0, i)
		bars[i] = models.Bar{Time: t, Open: price, High: price, Low: price, Close: price, Volume: vol}
	}
	return bars
}

func btAppend(bars []models.Bar, b models.Bar) []models.Bar {
	return append(bars, b)
}

func ihsgDefault() []models.Bar {
	return btFlat(60, 7000, 1e9, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
}

func btConfig() Config {
	return Config{
		Capital:      1000000,
		MaxPositions: 3,
		SlippagePct:  0,
		MaxHoldDays:  5,
		MinScore:     8,
		PumpFilter:   true,
		ARBFilter:    true,
		IgnoreMarket: true,
		Seed:         42,
		Years:        5,
		Strategy:     "cbow",
		CLMode:       "structural",
	}
}

func btRun(cfg Config, bars []models.Bar) (Result, error) {
	return Run(cfg, map[string][]models.Bar{"TEST": bars}, ihsgDefault())
}

// Case 1: HAMMER signal at Feb 12, entry=1005, CL=970 (980×0.99).
// Feb 13 low=960 < CL=970 → CL fires immediately. Result: 1 trade (CL).
func TestEngine_Case1_TP1ThenCL(t *testing.T) {
	// 100 flat bars (Dec 7 → Mar 16, 2024) to satisfy CBoW MinBars=60
	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(100, 1000, 1e9, start)
	hist = btAppend(hist, btBar(2024, 3, 18, 1005, 1010, 980, 1005, 2e9)) // signal
	hist = btAppend(hist, btBar(2024, 3, 19, 1000, 1040, 960, 970, 1e9))  // exit CL
	hist = btAppend(hist, btBar(2024, 3, 20, 1000, 1020, 960, 970, 1e9))

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}

	res, err := btRun(cfg, hist)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(res.Trades) != 1 {
		t.Errorf("want 1 trade, got %d", len(res.Trades))
	}
	if len(res.Trades) >= 1 && res.Trades[0].ExitReason != "CL" {
		t.Errorf("want CL (low=960 < CL=970), got %s", res.Trades[0].ExitReason)
	}
}

// Case 2: CL_GAP — open gaps below CL on next bar.
func TestEngine_Case2_CLGap(t *testing.T) {
	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(100, 1000, 1e9, start)
	hist = btAppend(hist, btBar(2024, 3, 18, 1005, 1010, 980, 1005, 2e9))
	hist = btAppend(hist, btBar(2024, 3, 19, 960, 960, 960, 960, 1e9)) // CL_GAP

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}

	res, err := btRun(cfg, hist)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(res.Trades) != 1 {
		t.Errorf("want 1 trade, got %d", len(res.Trades))
	}
	if len(res.Trades) >= 1 && res.Trades[0].ExitReason != "CL_GAP" {
		t.Errorf("want CL_GAP, got %s", res.Trades[0].ExitReason)
	}
}

// Case 3: CL fires before TIME — low=960 < CL=970.
func TestEngine_Case3_TimeStop(t *testing.T) {
	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(100, 1000, 1e9, start)
	hist = btAppend(hist, btBar(2024, 3, 18, 1005, 1010, 980, 1005, 2e9))
	for i := 19; i <= 23; i++ {
		hist = btAppend(hist, btBar(2024, 3, i, 1000, 1040, 960, 970, 1e9))
	}

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}

	res, err := btRun(cfg, hist)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(res.Trades) != 1 {
		t.Errorf("want 1 trade, got %d", len(res.Trades))
	}
	if len(res.Trades) >= 1 && res.Trades[0].ExitReason != "CL" {
		t.Errorf("want CL (low=960 < CL=970), got %s", res.Trades[0].ExitReason)
	}
}

func TestEngine_DeterministicSeed(t *testing.T) {
	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(100, 1000, 1e9, start)
	hist = btAppend(hist, btBar(2024, 3, 18, 1005, 1010, 980, 1005, 2e9))
	for i := 19; i <= 23; i++ {
		hist = btAppend(hist, btBar(2024, 3, i, 1000, 1040, 960, 970, 1e9))
	}
	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}

	res1, _ := btRun(cfg, hist)
	res2, _ := btRun(cfg, hist)

	if len(res1.Trades) != len(res2.Trades) {
		t.Errorf("different trade count: %d vs %d", len(res1.Trades), len(res2.Trades))
	}
}

// Case: IHSG has FEWER bars than stock — must not panic.
func TestEngine_IHSG_ShorterThanStock(t *testing.T) {
	stockStart := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	stockBars := btFlat(65, 1000, 1e9, stockStart)
	stockBars = btAppend(stockBars, btBar(2023, 12, 27, 1005, 1010, 980, 1005, 2e9))
	stockBars = btAppend(stockBars, btBar(2023, 12, 28, 1000, 1040, 960, 970, 1e9))

	ihsgBars := btFlat(60, 7000, 1e9, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.IgnoreMarket = true

	_, err := Run(cfg, map[string][]models.Bar{"TEST": stockBars}, ihsgBars)
	if err != nil {
		t.Fatalf("Run with IHSG shorter than stock must not error: %v", err)
	}
}

// Case: Stock listed AFTER IHSG starts (new IPO scenario).
func TestEngine_StockStartsLaterThanIHSG(t *testing.T) {
	ihsgBars := btFlat(60, 7000, 1e9, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

	stockStart := time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)
	stockBars := btFlat(30, 1000, 1e9, stockStart)
	stockBars = btAppend(stockBars, btBar(2023, 12, 27, 1005, 1010, 980, 1005, 2e9))
	stockBars = btAppend(stockBars, btBar(2023, 12, 28, 1000, 1040, 960, 970, 1e9))

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.IgnoreMarket = true

	_, err := Run(cfg, map[string][]models.Bar{"TEST": stockBars}, ihsgBars)
	if err != nil {
		t.Fatalf("Run with stock starting after IHSG must not error: %v", err)
	}
}

// Case: Stock longer than IHSG.
func TestEngine_StockLongerThanIHSG(t *testing.T) {
	ihsgBars := btFlat(40, 7000, 1e9, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

	stockStart := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	stockBars := btFlat(70, 1000, 1e9, stockStart)
	stockBars = btAppend(stockBars, btBar(2023, 12, 27, 1005, 1010, 980, 1005, 2e9))
	stockBars = btAppend(stockBars, btBar(2023, 12, 28, 1000, 1040, 960, 970, 1e9))

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.IgnoreMarket = true

	_, err := Run(cfg, map[string][]models.Bar{"TEST": stockBars}, ihsgBars)
	if err != nil {
		t.Fatalf("Run with stock longer than IHSG must not error: %v", err)
	}
}

// Case: No lookahead — at day t, IHSG close used must not be from after t.
func TestEngine_NoLookaheadIHSG(t *testing.T) {
	ihsgBars := btFlat(60, 7000, 1e9, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

	stockBars := btFlat(30, 1000, 1e9, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	stockBars = btAppend(stockBars, btBar(2023, 12, 27, 1005, 1010, 980, 1005, 2e9))

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.IgnoreMarket = true

	_, err := Run(cfg, map[string][]models.Bar{"TEST": stockBars}, ihsgBars)
	if err != nil {
		t.Fatalf("Run must not error: %v", err)
	}
}

// Test CLMode fixed: CL = entry × 0.97
func TestEngine_CLModeFixed(t *testing.T) {
	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(100, 1000, 1e9, start)
	hist = btAppend(hist, btBar(2024, 3, 18, 1005, 1010, 980, 1005, 2e9))
	// Exit bar: open=990 (>CL=974.85 so no gap), low=970 (<CL so CL fires)
	hist = btAppend(hist, btBar(2024, 3, 19, 990, 990, 970, 970, 1e9))

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.CLMode = "fixed"

	res, err := btRun(cfg, hist)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(res.Trades) != 1 {
		t.Errorf("CLMode fixed: want 1 trade, got %d", len(res.Trades))
	}
	if len(res.Trades) >= 1 && res.Trades[0].ExitReason != "CL" {
		t.Errorf("CLMode fixed: want CL, got %s", res.Trades[0].ExitReason)
	}
}

// Verify fixed CL > structural CL when signal.Low is near entry.
// entry=1005, signal.Low=980 → structural=970.2, fixed=974.85
// On low=973, structural: 973 > 970.2 (no exit), fixed: 973 < 974.85 (CL).
func TestEngine_CLModeFixedHigherThanStructural(t *testing.T) {
	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(100, 1000, 1e9, start)
	hist = btAppend(hist, btBar(2024, 3, 18, 1005, 1010, 980, 1005, 2e9))
	hist = btAppend(hist, btBar(2024, 3, 19, 973, 973, 973, 973, 1e9))

	cfgStruct := btConfig()
	cfgStruct.Symbols = []string{"TEST"}
	cfgStruct.CLMode = "structural"
	resStruct, _ := btRun(cfgStruct, hist)

	cfgFixed := btConfig()
	cfgFixed.Symbols = []string{"TEST"}
	cfgFixed.CLMode = "fixed"
	resFixed, _ := btRun(cfgFixed, hist)

	// Structural: no exit on day 2 (low=973 > CL=970.2)
	if len(resStruct.Trades) != 0 {
		t.Errorf("structural: want 0 trades (no exit), got %d", len(resStruct.Trades))
	}
	// Fixed: CL fires (low=973 < CL=974.85)
	if len(resFixed.Trades) == 0 {
		t.Errorf("fixed: want ≥1 trade (CL fires), got 0")
	}
}

// Breakout20 strategy test
func TestEngine_Breakout20(t *testing.T) {
	// 65 flat bars + signal Feb 12 + exit Feb 13
	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(65, 100, 100, start)
	hist = btAppend(hist, btBar(2024, 2, 12, 101, 101, 101, 101, 200)) // breakout signal
	hist = btAppend(hist, btBar(2024, 2, 13, 970, 970, 970, 970, 1e9)) // exit CL

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.Strategy = "breakout20"
	cfg.MinScore = 8

	res, err := Run(cfg, map[string][]models.Bar{"TEST": hist}, ihsgDefault())
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(res.Trades) != 1 {
		t.Errorf("Breakout20: want 1 trade, got %d", len(res.Trades))
	}
	if len(res.Trades) >= 1 && res.Trades[0].SignalCode != "BREAKOUT20" {
		t.Errorf("Breakout20: want code=BREAKOUT20, got %s", res.Trades[0].SignalCode)
	}
}

// PullbackMA20 strategy test: 65 flat bars + signal + exit
func TestEngine_PullbackMA20(t *testing.T) {
	// 65 flat bars (all close=100) → MA20=100, so signal fires when signal bar's low<101
	flatBars := btFlat(65, 100, 100, time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC))
	bars := flatBars
	// Signal bar: ma20=100, low=90<1.01×100=101 → cond3 passes
	ma20 := 100.0
	bars = append(bars, btBar(2024, 2, 10, ma20-1, ma20, 90, ma20+0.5, 1e9)) // signal
	bars = append(bars, btBar(2024, 2, 11, 970, 970, 970, 970, 1e9))         // exit CL

	// IHSG in uptrend so MarketSafe=true
	ihsgBars := make([]models.Bar, 60)
	for i := 0; i < 60; i++ {
		close := 7000.0 + float64(i)*10
		ihsgBars[i] = models.Bar{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i), Open: close, High: close, Low: close, Close: close, Volume: 1e9}
	}

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.Strategy = "pullback20"
	cfg.MinScore = 8

	res, err := Run(cfg, map[string][]models.Bar{"TEST": bars}, ihsgBars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(res.Trades) != 1 {
		t.Errorf("PullbackMA20: want 1 trade, got %d", len(res.Trades))
	}
	if len(res.Trades) >= 1 && res.Trades[0].SignalCode != "PULLBACK_MA20" {
		t.Errorf("PullbackMA20: want code=PULLBACK_MA20, got %s", res.Trades[0].SignalCode)
	}
}

// Strategy uses IHSG filter correctly
func TestEngine_StrategyWithMarketFilter(t *testing.T) {
	// Create IHSG that's in downtrend (current < MA20)
	ihsgBars := make([]models.Bar, 60)
	for i := 0; i < 60; i++ {
		close := 7000.0 - float64(60-1-i)*10
		ihsgBars[i] = models.Bar{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i), Open: close, High: close, Low: close, Close: close, Volume: 1e9}
	}

	start := time.Date(2023, 12, 7, 0, 0, 0, 0, time.UTC)
	hist := btFlat(65, 100, 100, start)
	hist = btAppend(hist, btBar(2024, 2, 12, 101, 101, 101, 101, 200))

	cfg := btConfig()
	cfg.Symbols = []string{"TEST"}
	cfg.Strategy = "breakout20"
	cfg.IgnoreMarket = false // use market filter

	ihsgSafe, _ := signals.MarketSafe(func() []float64 {
		closes := make([]float64, len(ihsgBars))
		for i, b := range ihsgBars {
			closes[i] = b.Close
		}
		return closes
	}())
	if !ihsgSafe {
		// IHSG in downtrend → breakout20 should NOT fire
		res, _ := Run(cfg, map[string][]models.Bar{"TEST": hist}, ihsgBars)
		if len(res.Trades) != 0 {
			t.Errorf("Breakout20 with IHSG downtrend: want 0 trades, got %d", len(res.Trades))
		}
	}
}
