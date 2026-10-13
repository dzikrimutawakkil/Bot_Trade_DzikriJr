package rules

import (
	"math"
	"testing"

	"learn-go/internal/models"
)

func TestEvaluate(t *testing.T) {
	defaultRules := Default
	// Base plan: entry 535, CL 515, TP1 555, TP1Done=false
	makePlan := func(highest float64, tp1Done bool) models.TradingPlan {
		return models.TradingPlan{
			EntryPrice:   535,
			TakeProfit:   555,
			CutLoss:      515,
			HighestPrice: highest,
			TP1Done:      tp1Done,
		}
	}

	cases := []struct {
		name     string
		plan     models.TradingPlan
		price    float64
		expected Decision
	}{
		{"CutLoss at boundary", makePlan(535, false), 514, CutLoss},
		{"CutLoss exactly at boundary", makePlan(535, false), 515, CutLoss},
		{"None above CL", makePlan(535, false), 516, None},
		{"TakeProfit1", makePlan(535, false), 555, TakeProfit1},
		{"TakeProfit1 skipped when TP1Done", makePlan(535, true), 555, None},
		{"TrailingStop (TSL=540, highest=556)", makePlan(556, false), 540, TrailingStop},
		{"None just above TSL", makePlan(556, false), 541, None},
		{"None below activation threshold", makePlan(545, false), 530, None},
		{"TP1Done clears TakeProfit trigger", makePlan(560, true), 560, None},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := defaultRules.Evaluate(c.plan, c.price)
			if got != c.expected {
				t.Errorf("Evaluate() = %v; want %v", got, c.expected)
			}
		})
	}
}

func TestTrailingActive(t *testing.T) {
	r := Default
	cases := []struct {
		name     string
		plan     models.TradingPlan
		expected bool
	}{
		{"highest=entry, net<2%", models.TradingPlan{EntryPrice: 535, HighestPrice: 535}, false},
		{"highest=555, net~2.91% > 2%", models.TradingPlan{EntryPrice: 535, HighestPrice: 555}, true},
		{"highest=545, net~1.46% < 2%", models.TradingPlan{EntryPrice: 535, HighestPrice: 545}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := r.TrailingActive(c.plan)
			if got != c.expected {
				t.Errorf("TrailingActive() = %v; want %v", got, c.expected)
			}
		})
	}
}

func TestSuggestPlan(t *testing.T) {
	r := Default
	// Base input: price 535, low 520, totalCapital 1M, inUse 0, safe
	baseIn := PlanInput{
		CurrentPrice: 535,
		SignalLow:    520,
		TotalCapital: 1000000,
		CapitalInUse: 0,
		MarketSafe:   true,
	}

	cases := []struct {
		name  string
		in    PlanInput
		check func(PlanOutput) bool
	}{
		{
			"price 535, low 520, inUse 0, safe",
			baseIn,
			func(o PlanOutput) bool {
				return o.CutLoss == 515 &&
					o.BuyMin == 530 &&
					o.BuyMax == 535 &&
					o.TP1 == 555 &&
					o.LossPerLot == 2000 &&
					o.MaxLotsByRisk == 25 &&
					o.MaxLotsByCash == 18 &&
					o.MaxLots == 18
			},
		},
		{
			"price 535, low 520, inUse 500k",
			PlanInput{CurrentPrice: 535, SignalLow: 520, TotalCapital: 1000000, CapitalInUse: 500000, MarketSafe: true},
			func(o PlanOutput) bool {
				return o.MaxLotsByCash == 9 && o.MaxLots == 9
			},
		},
		{
			"price 535, low 520, inUse 0, !safe",
			PlanInput{CurrentPrice: 535, SignalLow: 520, TotalCapital: 1000000, CapitalInUse: 0, MarketSafe: false},
			func(o PlanOutput) bool {
				return o.MaxLotsByRisk == 12 && o.MaxLots == 12
			},
		},
		{
			"CL at/above price → MaxLots=0",
			PlanInput{CurrentPrice: 535, SignalLow: 540, TotalCapital: 1000000, CapitalInUse: 0, MarketSafe: true},
			func(o PlanOutput) bool {
				return o.CutLoss == 535 && o.LossPerLot == 0 && o.MaxLots == 0 && len(o.Notes) > 0
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := r.SuggestPlan(c.in)
			if !c.check(out) {
				t.Errorf("SuggestPlan() = %+v; failed check", out)
			}
		})
	}
}

func TestUpdatePlan_KeyNotFound(t *testing.T) {
	// UpdatePlan test lives in internal/config/config_test.go
	// because it requires access to DataMutex and MyStocks from config.
	t.Skip("UpdatePlan tested in config_test.go")
}

func TestDefaultConstants(t *testing.T) {
	r := Default
	if r.TP1Pct != 4.0 {
		t.Errorf("TP1Pct = %v; want 4.0", r.TP1Pct)
	}
	if r.TrailingStopPct != 3.0 {
		t.Errorf("TrailingStopPct = %v; want 3.0", r.TrailingStopPct)
	}
	if r.TrailingActivationNetPct != 2.0 {
		t.Errorf("TrailingActivationNetPct = %v; want 2.0", r.TrailingActivationNetPct)
	}
	if r.BuyFee != 0.0015 || r.SellFee != 0.0025 {
		t.Errorf("BuyFee=%v/SellFee=%v; want 0.0015/0.0025", r.BuyFee, r.SellFee)
	}
}

func TestNetPNL(t *testing.T) {
	r := Default
	cases := []struct {
		entry, sell, expected, tol float64
	}{
		{535, 555, 3.32, 0.005},
		{535, 525, -2.26, 0.005},
		{3770, 3800, 0.39, 0.005},
	}
	for _, c := range cases {
		got := r.NetPNL(c.entry, c.sell)
		if math.Abs(got-c.expected) > c.tol {
			t.Errorf("NetPNL(%v, %v) = %v; want ~%v", c.entry, c.sell, got, c.expected)
		}
	}
}
