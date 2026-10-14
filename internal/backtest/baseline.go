package backtest

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"learn-go/internal/models"
	"learn-go/internal/pricing"
	"learn-go/internal/rules"
)

// BaselineResult is the outcome of a baseline run for compare.md.
type BaselineResult struct {
	Name           string
	FinalEquity    float64
	TotalInjected  float64
	ReturnPct      float64
	MaxDrawdownPct float64
	TradeCount     int
	ExpectancyPct  float64
	WinRate        float64
	ProfitFactor   float64
	AnnualizedPct  float64
	EquityCurve    []EquityPoint
	StartDate      time.Time
	EndDate        time.Time
}

// RunBaselineRandom3 — every week (Mon-like), pick `topN` symbols uniformly
// at random, equal weight, hold until next rebalance. Same exit rules as
// momentum-weekly (CL -3%, TP +5%, time 20d).
func RunBaselineRandom3(cfg Config, allBars map[string][]models.Bar, topN int) BaselineResult {
	return runWeeklyRebalancer(cfg, allBars, topN, "RANDOM3", true)
}

// RunBaselineDCA — every month, invest cfg.DCAMonthly split equally across
// the universe, then hold all positions until the end. No weekly churn.
// Proxy: equal-weight basket of all watchlist stocks.
func RunBaselineDCA(cfg Config, allBars map[string][]models.Bar) BaselineResult {
	tradingDays := collectTradingDays(allBars)
	if len(tradingDays) == 0 {
		return BaselineResult{Name: "DCA_NAIF"}
	}
	sort.Slice(tradingDays, func(i, j int) bool { return tradingDays[i].Before(tradingDays[j]) })

	cash := cfg.Capital
	totalInjected := cfg.Capital
	positions := make(map[string]*weeklyPosition)
	var prevMonth string
	equityCurve := make([]EquityPoint, 0, len(tradingDays))
	peak := cash

	// First day: deploy the initial capital equally across the universe.
	firstDay := tradingDays[0]
	for _, sym := range cfg.Symbols {
		bar := findBarAtDay(allBars[sym], firstDay)
		if bar == nil {
			continue
		}
		entry := pricing.RoundToFraction(bar.Open * (1 + cfg.SlippagePct/100))
		if entry <= 0 {
			continue
		}
		alloc := cash / float64(len(cfg.Symbols))
		costPerLot := entry * 100 * (1 + rules.Default.BuyFee)
		lot := int(math.Floor(alloc / costPerLot))
		if lot < 1 {
			continue
		}
		positions[sym] = &weeklyPosition{
			Symbol:     sym,
			EntryDate:  firstDay,
			EntryPrice: entry,
			Lot:        lot,
		}
		cash -= costPerLot * float64(lot)
	}

	for _, day := range tradingDays {
		monthKey := day.Format("2006-01")
		if monthKey != prevMonth {
			cash += cfg.DCAMonthly
			totalInjected += cfg.DCAMonthly
			// Reinvest fresh cash equally into all stocks.
			freePerStock := cfg.DCAMonthly / float64(len(cfg.Symbols))
			for _, sym := range cfg.Symbols {
				bar := findBarAtDay(allBars[sym], day)
				if bar == nil {
					continue
				}
				entry := pricing.RoundToFraction(bar.Open * (1 + cfg.SlippagePct/100))
				if entry <= 0 {
					continue
				}
				costPerLot := entry * 100 * (1 + rules.Default.BuyFee)
				lot := int(math.Floor(freePerStock / costPerLot))
				if lot < 1 {
					continue
				}
				pos, held := positions[sym]
				if held {
					// Add lots, weighted average price.
					totalLot := pos.Lot + lot
					pos.EntryPrice = (pos.EntryPrice*float64(pos.Lot) + entry*float64(lot)) / float64(totalLot)
					pos.Lot = totalLot
				} else {
					positions[sym] = &weeklyPosition{
						Symbol:     sym,
						EntryDate:  day,
						EntryPrice: entry,
						Lot:        lot,
					}
				}
				cash -= costPerLot * float64(lot)
			}
		}
		dayEquity := cash
		for _, pos := range positions {
			bar := findBarAtDay(allBars[pos.Symbol], day)
			if bar != nil {
				dayEquity += pos.EntryPrice * 100 * float64(pos.Lot) * (1 + rules.Default.NetPNL(pos.EntryPrice, bar.Close)/100)
			}
		}
		if dayEquity > peak {
			peak = dayEquity
		}
		dd := 0.0
		if peak > 0 {
			dd = (peak - dayEquity) / peak * 100
		}
		equityCurve = append(equityCurve, EquityPoint{Date: day, Equity: dayEquity, Drawdown: dd})
		prevMonth = monthKey
	}

	last := tradingDays[len(tradingDays)-1]
	finalCash := cash
	for _, pos := range positions {
		bar := findBarAtDay(allBars[pos.Symbol], last)
		if bar == nil {
			continue
		}
		pct := rules.Default.NetPNL(pos.EntryPrice, bar.Close)
		finalCash += pos.EntryPrice * 100 * float64(pos.Lot) * (1 + pct/100)
	}
	returnPct := 0.0
	if totalInjected > 0 {
		returnPct = (finalCash - totalInjected) / totalInjected * 100
	}
	maxDD := 0.0
	for _, p := range equityCurve {
		if p.Drawdown > maxDD {
			maxDD = p.Drawdown
		}
	}
	years := tradingDays[len(tradingDays)-1].Sub(tradingDays[0]).Hours() / 24 / 365.0
	annualized := 0.0
	if years > 0 {
		annualized = returnPct / years
	}
	return BaselineResult{
		Name:           "DCA_NAIF",
		FinalEquity:    finalCash,
		TotalInjected:  totalInjected,
		ReturnPct:      returnPct,
		MaxDrawdownPct: maxDD,
		AnnualizedPct:  annualized,
		EquityCurve:    equityCurve,
		StartDate:      tradingDays[0],
		EndDate:        last,
	}
}

// RunBaselineBuyHold — buy all watchlist stocks equal weight on day 1, hold
// until the end, ignore monthly injections after the initial basket.
func RunBaselineBuyHold(cfg Config, allBars map[string][]models.Bar) BaselineResult {
	tradingDays := collectTradingDays(allBars)
	if len(tradingDays) == 0 {
		return BaselineResult{Name: "BUY_HOLD"}
	}
	sort.Slice(tradingDays, func(i, j int) bool { return tradingDays[i].Before(tradingDays[j]) })

	cash := cfg.Capital
	totalInjected := cfg.Capital
	positions := make(map[string]*weeklyPosition)
	firstDay := tradingDays[0]
	for _, sym := range cfg.Symbols {
		bar := findBarAtDay(allBars[sym], firstDay)
		if bar == nil {
			continue
		}
		entry := pricing.RoundToFraction(bar.Open * (1 + cfg.SlippagePct/100))
		if entry <= 0 {
			continue
		}
		alloc := cash / float64(len(cfg.Symbols))
		costPerLot := entry * 100 * (1 + rules.Default.BuyFee)
		lot := int(math.Floor(alloc / costPerLot))
		if lot < 1 {
			continue
		}
		positions[sym] = &weeklyPosition{
			Symbol:     sym,
			EntryDate:  firstDay,
			EntryPrice: entry,
			Lot:        lot,
		}
		cash -= costPerLot * float64(lot)
	}

	equityCurve := make([]EquityPoint, 0, len(tradingDays))
	peak := cash
	for _, day := range tradingDays {
		dayEquity := cash
		for _, pos := range positions {
			bar := findBarAtDay(allBars[pos.Symbol], day)
			if bar != nil {
				dayEquity += pos.EntryPrice * 100 * float64(pos.Lot) * (1 + rules.Default.NetPNL(pos.EntryPrice, bar.Close)/100)
			}
		}
		if dayEquity > peak {
			peak = dayEquity
		}
		dd := 0.0
		if peak > 0 {
			dd = (peak - dayEquity) / peak * 100
		}
		equityCurve = append(equityCurve, EquityPoint{Date: day, Equity: dayEquity, Drawdown: dd})
	}

	last := tradingDays[len(tradingDays)-1]
	finalCash := cash
	for _, pos := range positions {
		bar := findBarAtDay(allBars[pos.Symbol], last)
		if bar == nil {
			continue
		}
		pct := rules.Default.NetPNL(pos.EntryPrice, bar.Close)
		finalCash += pos.EntryPrice * 100 * float64(pos.Lot) * (1 + pct/100)
	}
	returnPct := 0.0
	if totalInjected > 0 {
		returnPct = (finalCash - totalInjected) / totalInjected * 100
	}
	maxDD := 0.0
	for _, p := range equityCurve {
		if p.Drawdown > maxDD {
			maxDD = p.Drawdown
		}
	}
	years := last.Sub(tradingDays[0]).Hours() / 24 / 365.0
	annualized := 0.0
	if years > 0 {
		annualized = returnPct / years
	}
	return BaselineResult{
		Name:           "BUY_HOLD",
		FinalEquity:    finalCash,
		TotalInjected:  totalInjected,
		ReturnPct:      returnPct,
		MaxDrawdownPct: maxDD,
		AnnualizedPct:  annualized,
		EquityCurve:    equityCurve,
		StartDate:      tradingDays[0],
		EndDate:        last,
	}
}

func runWeeklyRebalancer(cfg Config, allBars map[string][]models.Bar, topN int, code string, randomize bool) BaselineResult {
	tradingDays := collectTradingDays(allBars)
	if len(tradingDays) == 0 {
		return BaselineResult{Name: code}
	}
	sort.Slice(tradingDays, func(i, j int) bool { return tradingDays[i].Before(tradingDays[j]) })

	clPct := cfg.CLFixedPct
	if clPct == 0 {
		clPct = 3.0
	}
	tpPct := cfg.TPPct
	if tpPct == 0 {
		tpPct = 5.0
	}
	maxHold := cfg.MaxHoldDays
	if maxHold == 0 {
		maxHold = 20
	}
	if topN == 0 {
		topN = cfg.MaxPositions
	}
	if topN == 0 {
		topN = 3
	}

	cash := cfg.Capital
	totalInjected := cfg.Capital
	positions := make(map[string]*weeklyPosition)
	order := make([]string, 0)
	trades := []Trade{}
	equityCurve := make([]EquityPoint, 0, len(tradingDays))
	peak := cash
	var prevWeek string
	var prevMonth string

	rng := rand.New(rand.NewSource(cfg.Seed))

	for _, day := range tradingDays {
		weekKey := day.Format("2006-W02")
		monthKey := day.Format("2006-01")

		if monthKey != prevMonth {
			cash += cfg.DCAMonthly
			totalInjected += cfg.DCAMonthly
		}

		if weekKey != prevWeek {
			// Exit all positions at the rebalance boundary (close any that
			// would otherwise linger, regardless of P/L).
			for _, sym := range order {
				pos := positions[sym]
				bar := findBarAtDay(allBars[sym], day)
				if bar == nil {
					continue
				}
				pct := rules.Default.NetPNL(pos.EntryPrice, bar.Open)
				posValue := pos.EntryPrice * 100 * float64(pos.Lot)
				pnl := posValue * pct / 100
				cash += posValue + pnl
				trades = append(trades, newTradeFromWeekly(*pos, day, "REBAL", pos.Lot, pct, pnl))
				delete(positions, sym)
			}
			order = order[:0]

			var pick []string
			if randomize {
				idx := rng.Perm(len(cfg.Symbols))
				for i := 0; i < topN && i < len(idx); i++ {
					pick = append(pick, cfg.Symbols[idx[i]])
				}
			} else {
				pick = cfg.Symbols
				if len(pick) > topN {
					pick = pick[:topN]
				}
			}

			openSlots := topN
			freeCash := cash / float64(topN)
			for _, sym := range pick {
				if openSlots <= 0 {
					break
				}
				bar := findBarAtDay(allBars[sym], day)
				if bar == nil {
					continue
				}
				entry := pricing.RoundToFraction(bar.Open * (1 + cfg.SlippagePct/100))
				if entry <= 0 {
					continue
				}
				costPerLot := entry * 100 * (1 + rules.Default.BuyFee)
				lot := int(math.Floor(freeCash / costPerLot))
				if lot < 1 {
					continue
				}
				cl := pricing.RoundToFraction(entry * (1 - clPct/100))
				tp := pricing.RoundToFraction(entry * (1 + tpPct/100))
				positions[sym] = &weeklyPosition{
					Symbol:     sym,
					EntryDate:  day,
					EntryPrice: entry,
					CL:         cl,
					TP:         tp,
					Lot:        lot,
					HoldDays:   0,
					SigCode:    code,
				}
				order = append(order, sym)
				cash -= costPerLot * float64(lot)
				openSlots--
			}
		}

		for _, sym := range order {
			pos := positions[sym]
			if pos == nil {
				continue
			}
			bar := findBarAtDay(allBars[sym], day)
			if bar == nil {
				continue
			}
			pos.HoldDays++

			posValue := pos.EntryPrice * 100 * float64(pos.Lot)

			if bar.Open <= pos.CL {
				pct := rules.Default.NetPNL(pos.EntryPrice, bar.Open)
				pnl := posValue * pct / 100
				cash += posValue + pnl
				trades = append(trades, newTradeFromWeekly(*pos, day, "CL", pos.Lot, pct, pnl))
				delete(positions, sym)
				order = removeString(order, sym)
				continue
			}
			if bar.Low <= pos.CL {
				pct := rules.Default.NetPNL(pos.EntryPrice, pos.CL)
				pnl := posValue * pct / 100
				cash += posValue + pnl
				trades = append(trades, newTradeFromWeekly(*pos, day, "CL", pos.Lot, pct, pnl))
				delete(positions, sym)
				order = removeString(order, sym)
				continue
			}
			if bar.High >= pos.TP {
				exitPrice := math.Max(bar.Open, pos.TP)
				pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
				pnl := posValue * pct / 100
				cash += posValue + pnl
				trades = append(trades, newTradeFromWeekly(*pos, day, "TP", pos.Lot, pct, pnl))
				delete(positions, sym)
				order = removeString(order, sym)
				continue
			}
			if pos.HoldDays >= maxHold {
				pct := rules.Default.NetPNL(pos.EntryPrice, bar.Close)
				pnl := posValue * pct / 100
				cash += posValue + pnl
				trades = append(trades, newTradeFromWeekly(*pos, day, "TIME", pos.Lot, pct, pnl))
				delete(positions, sym)
				order = removeString(order, sym)
			}
		}

		dayEquity := cash
		for _, sym := range order {
			pos := positions[sym]
			if pos == nil {
				continue
			}
			bar := findBarAtDay(allBars[sym], day)
			if bar != nil {
				dayEquity += pos.EntryPrice * 100 * float64(pos.Lot) * (1 + rules.Default.NetPNL(pos.EntryPrice, bar.Close)/100)
			}
		}
		if dayEquity > peak {
			peak = dayEquity
		}
		dd := 0.0
		if peak > 0 {
			dd = (peak - dayEquity) / peak * 100
		}
		equityCurve = append(equityCurve, EquityPoint{Date: day, Equity: dayEquity, Drawdown: dd})

		prevWeek = weekKey
		prevMonth = monthKey
	}

	last := tradingDays[len(tradingDays)-1]
	for _, sym := range order {
		pos := positions[sym]
		if pos == nil {
			continue
		}
		bar := findBarAtDay(allBars[sym], last)
		if bar == nil {
			continue
		}
		pct := rules.Default.NetPNL(pos.EntryPrice, bar.Close)
		posValue := pos.EntryPrice * 100 * float64(pos.Lot)
		pnl := posValue * pct / 100
		cash += posValue + pnl
		trades = append(trades, newTradeFromWeekly(*pos, last, "EOD_CLOSE", pos.Lot, pct, pnl))
	}

	metrics := calcMetrics(trades)
	returnPct := 0.0
	if totalInjected > 0 {
		returnPct = (cash - totalInjected) / totalInjected * 100
	}
	maxDD := 0.0
	for _, p := range equityCurve {
		if p.Drawdown > maxDD {
			maxDD = p.Drawdown
		}
	}
	years := last.Sub(tradingDays[0]).Hours() / 24 / 365.0
	annualized := 0.0
	if years > 0 {
		annualized = returnPct / years
	}
	return BaselineResult{
		Name:           code,
		FinalEquity:    cash,
		TotalInjected:  totalInjected,
		ReturnPct:      returnPct,
		MaxDrawdownPct: maxDD,
		TradeCount:     len(trades),
		ExpectancyPct:  metrics.ExpectancyPct,
		WinRate:        metrics.WinRate,
		ProfitFactor:   metrics.ProfitFactor,
		AnnualizedPct:  annualized,
		EquityCurve:    equityCurve,
		StartDate:      tradingDays[0],
		EndDate:        last,
	}
}
