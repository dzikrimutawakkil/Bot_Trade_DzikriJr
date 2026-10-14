package backtest

import (
	"fmt"
	"math"
	"sort"
	"time"

	"learn-go/internal/models"
	"learn-go/internal/pricing"
	"learn-go/internal/rules"
	"learn-go/internal/signals"
)

type Config struct {
	Symbols      []string
	Capital      float64
	MaxPositions int
	SlippagePct  float64
	MaxHoldDays  int
	MinScore     float64
	PumpFilter   bool
	ARBFilter    bool
	IgnoreMarket bool
	OOSFrom      time.Time
	Seed         int64
	OutDir       string
	Refresh      bool
	Years        int
	Strategy     string
	CLMode       string

	// Weekly momentum rotation only
	DCAMonthly float64
	CLFixedPct float64
	TPPct      float64
}

type Trade struct {
	EntryDate       time.Time
	Symbol          string
	SignalCode      string
	EntryPrice      float64
	ExitDate        time.Time
	ExitReason      string
	LotsSold        int
	LotsTotal       int
	PNLPctNet       float64
	PNLRpNet        float64
	HoldDays        int
	HoldTradingDays int
}

type EquityPoint struct {
	Date     time.Time
	Equity   float64
	Drawdown float64
}

type Result struct {
	Trades              []Trade
	EquityCurve         []EquityPoint
	Metrics             Metrics
	SignalCounts        map[string]int
	IHSGFiltered        int
	FinalCapital        float64
	TotalReturnPct      float64
	SkippedSignals      []SkippedSignal
	SignalCountsPerMode map[string]map[string]int
}

type Metrics struct {
	TotalTrades     int
	WinRate         float64
	AvgWinPct       float64
	AvgLossPct      float64
	ExpectancyPct   float64
	ExpectancyRp    float64
	ProfitFactor    float64
	ProfitFactorPct float64
	MaxDrawdown     float64
	AvgHoldDays     float64
	ExitReasons     map[string]ExitStats
}

type ExitStats struct {
	Count     int
	AvgPNLPct float64
}

type SkippedSignal struct {
	Date       time.Time
	Symbol     string
	SignalCode string
	EntryPrice float64
	Alloc      float64
	CostPerLot float64
}

type position struct {
	Symbol     string
	EntryDate  time.Time
	EntryPrice float64
	CL         float64
	TP1        float64
	TP1Done    bool
	Lot        int
	Highest    float64
	HoldDays   int
	SigCode    string
}

func resolveStrategy(name string) signals.Strategy {
	if name == "" || name == "cbow" {
		return &signals.CBoWStrategy{}
	}
	s, err := signals.GetStrategy(name)
	if err != nil {
		return &signals.CBoWStrategy{}
	}
	return s
}

func effectiveCL(clMode string, sig signals.Signal, entry float64) float64 {
	if clMode == "fixed" {
		return rules.Default.DefaultCutLoss(entry)
	}
	return rules.Default.StructuralCutLoss(sig.Low)
}

func Run(cfg Config, allBars map[string][]models.Bar, ihsgBars []models.Bar) (Result, error) {
	ihsgCloses := make([]float64, 0, len(ihsgBars))
	ihsgDates := make([]time.Time, 0, len(ihsgBars))
	for _, b := range ihsgBars {
		ihsgCloses = append(ihsgCloses, b.Close)
		ihsgDates = append(ihsgDates, b.Time)
	}

	ihsgDateMap := make(map[string]int)
	for i, b := range ihsgBars {
		ihsgDateMap[b.Time.Format("2006-01-02")] = i
	}

	ihsgClosesUpTo := func(day time.Time) []float64 {
		dayStr := day.Format("2006-01-02")
		idx := -1
		if pos, ok := ihsgDateMap[dayStr]; ok {
			idx = pos
		} else {
			for i := len(ihsgDates) - 1; i >= 0; i-- {
				if ihsgDates[i].Format("2006-01-02") <= dayStr {
					idx = i
					break
				}
			}
		}
		if idx < 0 {
			return ihsgCloses
		}
		return ihsgCloses[:idx+1]
	}

	tradingDays := collectTradingDays(allBars)
	if len(tradingDays) == 0 {
		return Result{}, fmt.Errorf("no trading days found")
	}
	sort.Slice(tradingDays, func(i, j int) bool { return tradingDays[i].Before(tradingDays[j]) })

	oosIdx := len(tradingDays) * 70 / 100
	if cfg.OOSFrom.After(tradingDays[0]) {
		for i, d := range tradingDays {
			if !d.Before(cfg.OOSFrom) {
				oosIdx = i
				break
			}
		}
	}
	_ = oosIdx

	maxHold := cfg.MaxHoldDays
	if maxHold == 0 {
		maxHold = 5
	}

	strategy := resolveStrategy(cfg.Strategy)
	clMode := cfg.CLMode
	if clMode == "" {
		clMode = "structural"
	}

	findBarIdx := func(bars []models.Bar, day time.Time) int {
		dayStr := day.Format("2006-01-02")
		for i, b := range bars {
			if b.Time.Format("2006-01-02") == dayStr {
				return i
			}
		}
		return -1
	}

	positions := []position{}
	trades := []Trade{}
	equity := cfg.Capital
	peakEquity := equity

	signalCounts := make(map[string]int)
	ihsgFiltered := 0
	skippedSignals := []SkippedSignal{}

	for dayIdx := range tradingDays {
		day := tradingDays[dayIdx]

		for _, bars := range allBars {
			if len(bars) == 0 {
				continue
			}
			barIdx := findBarIdx(bars, day)
			if barIdx < 0 {
				continue
			}
			if barIdx < strategy.MinBars()-1 {
				continue
			}
			ihsgSafe, _ := signals.MarketSafe(ihsgClosesUpTo(day))
			sig := strategy.Evaluate(bars[:barIdx+1], ihsgSafe)
			signalCounts[sig.Code]++
			if !cfg.IgnoreMarket && ihsgSafe {
				ihsgFiltered++
			}
		}

		for pi := 0; pi < len(positions); {
			pos := &positions[pi]
			bar := findBarAtDay(allBars[pos.Symbol], day)
			if bar == nil {
				pi++
				continue
			}

			pos.HoldDays++
			if bar.High > pos.Highest {
				pos.Highest = bar.High
			}

			dayPNL := 0.0
			liquidated := false

			if bar.Open <= pos.CL {
				exitPrice := bar.Open
				exitReason := "CL_GAP"
				pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
				posValue := pos.EntryPrice * 100 * float64(pos.Lot)
				dayPNL = posValue * pct / 100
				trades = append(trades, newTrade(*pos, day, exitReason, pos.Lot, pos.Lot, pct, dayPNL))
				liquidated = true
			} else if bar.Low <= pos.CL {
				exitPrice := pos.CL
				exitReason := "CL"
				pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
				posValue := pos.EntryPrice * 100 * float64(pos.Lot)
				dayPNL = posValue * pct / 100
				trades = append(trades, newTrade(*pos, day, exitReason, pos.Lot, pos.Lot, pct, dayPNL))
				liquidated = true
			} else {
				tslActive := rules.Default.NetPNL(pos.EntryPrice, pos.Highest) >= rules.Default.TrailingActivationNetPct
				if tslActive {
					tsl := rules.Default.TrailingStopPrice(pos.Highest)
					if bar.Open <= tsl {
						exitPrice := bar.Open
						exitReason := "TSL_GAP"
						pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
						posValue := pos.EntryPrice * 100 * float64(pos.Lot)
						dayPNL = posValue * pct / 100
						trades = append(trades, newTrade(*pos, day, exitReason, pos.Lot, pos.Lot, pct, dayPNL))
						liquidated = true
					} else if bar.Low <= tsl {
						exitPrice := tsl
						exitReason := "TSL"
						pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
						posValue := pos.EntryPrice * 100 * float64(pos.Lot)
						dayPNL = posValue * pct / 100
						trades = append(trades, newTrade(*pos, day, exitReason, pos.Lot, pos.Lot, pct, dayPNL))
						liquidated = true
					}
				}
			}

			if !liquidated && !pos.TP1Done && bar.High >= pos.TP1 {
				exitPrice := math.Max(bar.Open, pos.TP1)
				exitReason := "TP1"
				halfLot := int(math.Floor(float64(pos.Lot) * rules.Default.TP1SellFraction))
				if halfLot < 1 {
					halfLot = 1
				}
				if halfLot > pos.Lot {
					halfLot = pos.Lot
				}
				pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
				posValue := pos.EntryPrice * 100 * float64(halfLot)
				dayPNL = posValue * pct / 100
				trades = append(trades, newTrade(*pos, day, exitReason, halfLot, pos.Lot, pct, dayPNL))
				pos.Lot -= halfLot
				pos.TP1Done = true
				equity += dayPNL
				if pos.Lot == 0 {
					positions = append(positions[:pi], positions[pi+1:]...)
					continue
				}
			}

			if !liquidated && pos.HoldDays >= maxHold {
				exitPrice := bar.Close
				exitReason := "TIME"
				pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
				posValue := pos.EntryPrice * 100 * float64(pos.Lot)
				dayPNL = posValue * pct / 100
				trades = append(trades, newTrade(*pos, day, exitReason, pos.Lot, pos.Lot, pct, dayPNL))
				liquidated = true
			}

			if liquidated {
				equity += dayPNL
				positions = append(positions[:pi], positions[pi+1:]...)
				continue
			}
			pi++
		}

		if len(positions) < cfg.MaxPositions && equity > 250000 {
			for _, sym := range cfg.Symbols {
				if len(positions) >= cfg.MaxPositions {
					break
				}
				alreadyHeld := false
				for _, p := range positions {
					if p.Symbol == sym {
						alreadyHeld = true
						break
					}
				}
				if alreadyHeld {
					continue
				}

				bar := findBarAtDay(allBars[sym], day)
				if bar == nil {
					continue
				}

				barIdx := findBarIdx(allBars[sym], day)
				if barIdx < strategy.MinBars()-1 {
					continue
				}

				ihsgSafe, _ := signals.MarketSafe(ihsgClosesUpTo(day))
				sig := strategy.Evaluate(allBars[sym][:barIdx+1], ihsgSafe)

				if sig.Score < cfg.MinScore {
					continue
				}

				if !cfg.IgnoreMarket && !ihsgSafe {
					continue
				}

				entryPrice := pricing.RoundToFraction(bar.Close * (1 + cfg.SlippagePct/100))
				alloc := equity / float64(cfg.MaxPositions)
				costPerLot := entryPrice * 100 * (1 + rules.Default.BuyFee)
				lot := int(math.Floor(alloc / costPerLot))
				if lot < 1 {
					skippedSignals = append(skippedSignals, SkippedSignal{
						Date:       day,
						Symbol:     sym,
						SignalCode: sig.Code,
						EntryPrice: entryPrice,
						Alloc:      alloc,
						CostPerLot: costPerLot,
					})
					continue
				}

				cl := effectiveCL(clMode, sig, entryPrice)
				tp1 := rules.Default.TP1Price(entryPrice)
				positions = append(positions, position{
					Symbol:     sym,
					EntryDate:  day,
					EntryPrice: entryPrice,
					CL:         cl,
					TP1:        tp1,
					TP1Done:    false,
					Lot:        lot,
					Highest:    entryPrice,
					HoldDays:   0,
					SigCode:    sig.Code,
				})
			}
		}

		dayEquity := equity
		for _, pos := range positions {
			posBar := findBarAtDay(allBars[pos.Symbol], day)
			if posBar != nil {
				dayEquity += pos.EntryPrice * 100 * float64(pos.Lot) * rules.Default.NetPNL(pos.EntryPrice, posBar.Close) / 100
			}
		}

		if dayEquity > peakEquity {
			peakEquity = dayEquity
		}
		dd := (peakEquity - dayEquity) / peakEquity * 100
		if dd < 0 {
			dd = 0
		}
		_ = dd
	}

	metrics := calcMetrics(trades)
	finalCapital := equity
	for _, pos := range positions {
		bar := findBarAtDay(allBars[pos.Symbol], tradingDays[len(tradingDays)-1])
		if bar != nil {
			finalCapital += pos.EntryPrice * 100 * float64(pos.Lot) * rules.Default.NetPNL(pos.EntryPrice, bar.Close) / 100
		}
	}
	totalReturnPct := (finalCapital - cfg.Capital) / cfg.Capital * 100

	return Result{
		Trades:         trades,
		EquityCurve:    nil,
		Metrics:        metrics,
		SignalCounts:   signalCounts,
		IHSGFiltered:   ihsgFiltered,
		FinalCapital:   finalCapital,
		TotalReturnPct: totalReturnPct,
		SkippedSignals: skippedSignals,
	}, nil
}

func newTrade(pos position, day time.Time, reason string, lotsSold, lotsTotal int, pct, pnl float64) Trade {
	holdDays := int(day.Sub(pos.EntryDate).Hours() / 24)
	holdTrading := 0
	for d := pos.EntryDate.AddDate(0, 0, 1); d.Before(day) || d.Equal(day); d = d.AddDate(0, 0, 1) {
		w := d.Weekday()
		if w != time.Saturday && w != time.Sunday {
			holdTrading++
		}
		if d.Equal(day) {
			break
		}
	}
	return Trade{
		EntryDate:       pos.EntryDate,
		Symbol:          pos.Symbol,
		SignalCode:      pos.SigCode,
		EntryPrice:      pos.EntryPrice,
		ExitDate:        day,
		ExitReason:      reason,
		LotsSold:        lotsSold,
		LotsTotal:       lotsTotal,
		PNLPctNet:       pct,
		PNLRpNet:        pnl,
		HoldDays:        holdDays,
		HoldTradingDays: holdTrading,
	}
}

func collectTradingDays(allBars map[string][]models.Bar) []time.Time {
	daySet := make(map[string]bool)
	for _, bars := range allBars {
		for _, b := range bars {
			daySet[b.Time.Format("2006-01-02")] = true
		}
	}
	days := make([]time.Time, 0, len(daySet))
	for d := range daySet {
		t, _ := time.Parse("2006-01-02", d)
		days = append(days, t)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days
}

func findBarAtDay(bars []models.Bar, day time.Time) *models.Bar {
	dayStr := day.Format("2006-01-02")
	for _, b := range bars {
		if b.Time.Format("2006-01-02") == dayStr {
			return &b
		}
	}
	return nil
}

func calcMetrics(trades []Trade) Metrics {
	if len(trades) == 0 {
		return Metrics{}
	}
	m := Metrics{TotalTrades: len(trades), ExitReasons: make(map[string]ExitStats)}

	var sumWinRp, sumLossRp float64
	var winCount, lossCount, totalHold int

	for _, t := range trades {
		if t.PNLRpNet > 0 {
			sumWinRp += t.PNLRpNet
			winCount++
		} else {
			sumLossRp += -t.PNLRpNet
			lossCount++
		}
		totalHold += t.HoldTradingDays

		es := m.ExitReasons[t.ExitReason]
		es.Count++
		es.AvgPNLPct = (es.AvgPNLPct*float64(es.Count-1) + t.PNLPctNet) / float64(es.Count)
		m.ExitReasons[t.ExitReason] = es
	}

	m.WinRate = float64(winCount) / float64(len(trades)) * 100

	if winCount > 0 {
		m.AvgWinPct = sumWinRp / float64(winCount)
	}
	if lossCount > 0 {
		m.AvgLossPct = sumLossRp / float64(lossCount)
	}

	// Profit Factor (Rp): sum(win Rp) / |sum(loss Rp)|
	if sumLossRp > 0 {
		m.ProfitFactor = sumWinRp / sumLossRp
	} else if sumWinRp > 0 {
		m.ProfitFactor = sumWinRp
	}

	// Profit Factor (%): sum(pnl_pct_net > 0) / |sum(pnl_pct_net <= 0)|
	sumWinPct := 0.0
	sumLossPct := 0.0
	for _, t := range trades {
		if t.PNLPctNet > 0 {
			sumWinPct += t.PNLPctNet
		} else {
			sumLossPct += -t.PNLPctNet
		}
	}
	if sumLossPct > 0 {
		m.ProfitFactorPct = sumWinPct / sumLossPct
	} else if sumWinPct > 0 {
		m.ProfitFactorPct = sumWinPct
	}

	if len(trades) > 0 {
		m.AvgHoldDays = float64(totalHold) / float64(len(trades))
	}

	var avgWinPctD, avgLossPctD float64
	for _, t := range trades {
		if t.PNLRpNet > 0 {
			avgWinPctD += t.PNLPctNet
		} else {
			avgLossPctD += -t.PNLPctNet
		}
	}
	if winCount > 0 {
		avgWinPctD /= float64(winCount)
	}
	if lossCount > 0 {
		avgLossPctD /= float64(lossCount)
	}
	m.ExpectancyPct = (m.WinRate/100)*avgWinPctD - (1-m.WinRate/100)*avgLossPctD

	var totalPNL float64
	for _, t := range trades {
		totalPNL += t.PNLRpNet
	}
	if len(trades) > 0 {
		m.ExpectancyRp = totalPNL / float64(len(trades))
	}

	peak := 0.0
	maxDD := 0.0
	running := 0.0
	for _, t := range trades {
		running += t.PNLRpNet
		if running > peak {
			peak = running
		}
		dd := peak - running
		if dd > maxDD {
			maxDD = dd
		}
	}
	m.MaxDrawdown = maxDD

	return m
}
