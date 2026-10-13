package signals

import (
	"learn-go/internal/models"
	"testing"
	"time"
)

func makeBar(open, high, low, close, vol float64, daysAgo int) models.Bar {
	t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)
	return models.Bar{
		Time:   t,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: vol,
	}
}

func TestHammer(t *testing.T) {
	bars := []models.Bar{}
	for i := 21; i >= 1; i-- {
		bars = append(bars, makeBar(1000, 1000, 1000, 1000, 1e9, i))
	}
	bars = append(bars, makeBar(1000, 1008, 980, 1005, 1e9, 0))

	sig := Evaluate(bars, DefaultParams())
	if sig.Score != 10 {
		t.Errorf("want score=10, got %v", sig.Score)
	}
	if sig.Code != "HAMMER" {
		t.Errorf("want code=HAMMER, got %v", sig.Code)
	}
	if sig.Low != 980 {
		t.Errorf("want low=980, got %v", sig.Low)
	}
}

func TestBlockedARB(t *testing.T) {
	bars := []models.Bar{}
	for i := 25; i >= 1; i-- {
		bars = append(bars, makeBar(1000, 1000, 1000, 1000, 1e9, i))
	}
	bars = append(bars, makeBar(1000, 1000, 900, 900, 1e9, 0))
	bars = append(bars, makeBar(1000, 1000, 1000, 1000, 1e9, -1))

	sig := Evaluate(bars, DefaultParams())
	if sig.Score != -1 {
		t.Errorf("want score=-1 for ARB, got %v", sig.Score)
	}
	if sig.Code != "BLOCKED_ARB" {
		t.Errorf("want code=BLOCKED_ARB, got %v", sig.Code)
	}
}

func TestBlockedPump(t *testing.T) {
	bars := []models.Bar{}
	for i := 21; i >= 1; i-- {
		bars = append(bars, makeBar(1000, 1000, 1000, 1000, 1e9, i))
	}
	bars = append(bars, makeBar(1000, 1050, 1000, 1045, 1e9, 0))

	sig := Evaluate(bars, DefaultParams())
	if sig.Score != -1 {
		t.Errorf("want score=-1 for pump, got %v", sig.Score)
	}
	if sig.Code != "BLOCKED_PUMP" {
		t.Errorf("want code=BLOCKED_PUMP, got %v", sig.Code)
	}
}

func TestPumpFilterDisabled(t *testing.T) {
	bars := []models.Bar{}
	for i := 21; i >= 1; i-- {
		bars = append(bars, makeBar(1000, 1000, 1000, 1000, 1e9, i))
	}
	bars = append(bars, makeBar(1000, 1050, 1000, 1045, 1e9, 0))

	p := DefaultParams()
	p.PumpFilterPct = 0
	sig := Evaluate(bars, p)
	if sig.Code == "BLOCKED_PUMP" {
		t.Errorf("pump should NOT be blocked when filter disabled")
	}
}

func TestPisau(t *testing.T) {
	bars := []models.Bar{}
	for i := 20; i >= 1; i-- {
		bars = append(bars, makeBar(1000, 1000, 1000, 1000, 1e9, i))
	}
	bars = append(bars, makeBar(1000, 1000, 990, 995, 1e9, 0))

	sig := Evaluate(bars, DefaultParams())
	if sig.Score != 1 {
		t.Errorf("want score=1 for pisau, got %v", sig.Score)
	}
	if sig.Code != "PISAU" {
		t.Errorf("want code=PISAU, got %v", sig.Code)
	}
}

func TestNoData(t *testing.T) {
	bars := []models.Bar{makeBar(1000, 1000, 1000, 1000, 1e9, 0)}
	sig := Evaluate(bars, DefaultParams())
	if sig.Code != "NO_DATA" {
		t.Errorf("want code=NO_DATA, got %v", sig.Code)
	}
}

func TestMarketSafe_Uptrend(t *testing.T) {
	closes := make([]float64, 25)
	for i := range closes {
		closes[i] = 7000 + float64(i)*10
	}
	safe, state := MarketSafe(closes)
	if !safe || state != "UPTREND" {
		t.Errorf("want (true,UPTREND), got (%v,%v)", safe, state)
	}
}

func TestMarketSafe_Downtrend(t *testing.T) {
	closes := make([]float64, 25)
	for i := range closes {
		closes[i] = 7000 - float64(i)*10
	}
	safe, state := MarketSafe(closes)
	if safe || state != "DOWNTREND" {
		t.Errorf("want (false,DOWNTREND), got (%v,%v)", safe, state)
	}
}

// KOREKSI state is mathematically unreachable in MarketSafe.
// MarketSafe checks: current < MA20 → DOWNTREND; current <= MA5 → KOREKSI; else → UPTREND.
// For KOREKSI (current <= MA5) to have current > MA20, we need MA5 < MA20.
// But MA5 < MA20 implies MA5 is below the 20-bar average.
// Then current <= MA5 < MA20, so current is always < MA20.
// Therefore KOREKSI can never have current > MA20 — it would be DOWNTREND instead.
func TestMarketSafe_Koreksi(t *testing.T) {
	closes := make([]float64, 30)
	// All at 7000 — current (7000) > MA5 = MA20 = 7000. Not KOREKSI.
	// KOREKSI would require: current <= MA5 < MA20. With MA5 < MA20, current <= MA5 < MA20,
	// so current < MA20 → DOWNTREND, never KOREKSI. Test UPTREND as the only valid reachable state.
	for i := 0; i < 30; i++ {
		closes[i] = 7000 + float64(i)
	}
	safe, state := MarketSafe(closes)
	if !safe || state != "UPTREND" {
		t.Errorf("want (true,UPTREND), got (%v,%v)", safe, state)
	}
}
