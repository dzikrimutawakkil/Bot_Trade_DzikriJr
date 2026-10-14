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

type weeklyPosition struct {
	Symbol     string
	EntryDate  time.Time
	EntryPrice float64
	CL         float64
	TP         float64
	Lot        int
	HoldDays   int
	SigCode    string
}

// RunWeekly runs the Weekly Momentum Rotation backtest. Unlike Run() which
// is a daily signal-sweep, this:
//
//   - Rebalances on the first trading day of each ISO week (Monday-like).
//   - On rebalance: exits positions whose trend broke (MA5<MA20) or that
//     are no longer in the top-N candidates, then opens new positions for
//     any free slots using equal-weight of available cash.
//   - Injects fresh capital on the first trading day of each month.
//   - Tracks daily equity (cash + mark-to-market positions) and emits a
//     full equity curve so DCA impact is visible.
//
// Exit rules per spec: CL=-3% entry fixed, TP=+5% sell 100% lot, time stop
// 20 trading days, trend-break exit on weekly rebalance.
func RunWeekly(cfg Config, allBars map[string][]models.Bar) (Result, error) {
	tradingDays := collectTradingDays(allBars)
	if len(tradingDays) == 0 {
		return Result{}, fmt.Errorf("no trading days found")
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
	topN := cfg.MaxPositions
	if topN == 0 {
		topN = 3
	}

	positions := make(map[string]*weeklyPosition)
	order := make([]string, 0)
	trades := []Trade{}
	cash := cfg.Capital
	peakEquity := cash
	totalInjected := cfg.Capital
	equityCurve := make([]EquityPoint, 0, len(tradingDays))

	var prevWeekKey, prevMonthKey string
	weekRebalances := 0
	monthInjections := 0

	for _, day := range tradingDays {
		weekKey := day.Format("2006-W02")
		monthKey := day.Format("2006-01")
		isFirstDayOfWeek := weekKey != prevWeekKey
		isFirstDayOfMonth := monthKey != prevMonthKey

		if isFirstDayOfMonth {
			cash += cfg.DCAMonthly
			totalInjected += cfg.DCAMonthly
			monthInjections++
		}

		if isFirstDayOfWeek {
			weekRebalances++
			candidates := signals.SelectWeeklyCandidates(allBars, cfg.Symbols, day, topN)
			keep := make(map[string]bool, len(candidates))
			for _, c := range candidates {
				keep[c.Symbol] = true
			}

			var toClose []string
			for sym := range positions {
				trendBroken := signals.TrendBroken(allBars[sym], day)
				noLongerPicked := !keep[sym]

				if trendBroken || noLongerPicked {
					toClose = append(toClose, sym)
				}
			}

			for _, sym := range toClose {
				bar := findBarAtDay(allBars[sym], day)
				if bar == nil {
					delete(positions, sym)
					order = removeString(order, sym)
					continue
				}
				exitPrice := bar.Open
				pos := positions[sym]
				reason := "REBAL"
				if signals.TrendBroken(allBars[sym], day) {
					reason = "TREND_BROKEN"
				}
				pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
				posValue := pos.EntryPrice * 100 * float64(pos.Lot)
				pnl := posValue * pct / 100
				cash += posValue + pnl
				trades = append(trades, newTradeFromWeekly(*pos, day, reason, pos.Lot, pct, pnl))
				delete(positions, sym)
				order = removeString(order, sym)
			}

			openSlots := topN - len(positions)
			if openSlots > 0 && cash > 0 {
				freeCashPerSlot := cash / float64(openSlots)
				for _, c := range candidates {
					if openSlots <= 0 {
						break
					}
					if _, held := positions[c.Symbol]; held {
						continue
					}
					bar := findBarAtDay(allBars[c.Symbol], day)
					if bar == nil {
						continue
					}
					entryPrice := pricing.RoundToFraction(bar.Open * (1 + cfg.SlippagePct/100))
					if entryPrice <= 0 {
						continue
					}
					costPerLot := entryPrice * 100 * (1 + rules.Default.BuyFee)
					lot := int(math.Floor(freeCashPerSlot / costPerLot))
					if lot < 1 {
						continue
					}
					cl := pricing.RoundToFraction(entryPrice * (1 - clPct/100))
					tp := pricing.RoundToFraction(entryPrice * (1 + tpPct/100))
					positions[c.Symbol] = &weeklyPosition{
						Symbol:     c.Symbol,
						EntryDate:  day,
						EntryPrice: entryPrice,
						CL:         cl,
						TP:         tp,
						Lot:        lot,
						HoldDays:   0,
						SigCode:    "MOM_WEEKLY",
					}
					order = append(order, c.Symbol)
					cash -= costPerLot * float64(lot)
					openSlots--
				}
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
			bar := findBarAtDay(allBars[sym], day)
			if pos != nil && bar != nil {
				dayEquity += pos.EntryPrice * 100 * float64(pos.Lot) * (1 + rules.Default.NetPNL(pos.EntryPrice, bar.Close)/100)
			}
		}
		if dayEquity > peakEquity {
			peakEquity = dayEquity
		}
		dd := 0.0
		if peakEquity > 0 {
			dd = (peakEquity - dayEquity) / peakEquity * 100
		}
		equityCurve = append(equityCurve, EquityPoint{
			Date:     day,
			Equity:   dayEquity,
			Drawdown: dd,
		})

		prevWeekKey = weekKey
		prevMonthKey = monthKey
	}

	last := tradingDays[len(tradingDays)-1]
	for _, sym := range order {
		pos := positions[sym]
		if pos == nil {
			continue
		}
		bar := findBarAtDay(allBars[sym], last)
		exitPrice := 0.0
		if bar != nil {
			exitPrice = bar.Close
		}
		if exitPrice <= 0 {
			continue
		}
		posValue := pos.EntryPrice * 100 * float64(pos.Lot)
		pct := rules.Default.NetPNL(pos.EntryPrice, exitPrice)
		pnl := posValue * pct / 100
		cash += posValue + pnl
		trades = append(trades, newTradeFromWeekly(*pos, last, "EOD_CLOSE", pos.Lot, pct, pnl))
	}

	metrics := calcMetrics(trades)
	investedProfit := 0.0
	if totalInjected > 0 {
		investedProfit = (cash - totalInjected) / totalInjected * 100
	}

	signalCounts := map[string]int{
		"MOM_WEEKLY": len(trades),
	}

	_ = weekRebalances
	_ = monthInjections

	return Result{
		Trades:         trades,
		EquityCurve:    equityCurve,
		Metrics:        metrics,
		SignalCounts:   signalCounts,
		IHSGFiltered:   0,
		FinalCapital:   cash,
		TotalReturnPct: investedProfit,
		SkippedSignals: nil,
	}, nil
}

func newTradeFromWeekly(pos weeklyPosition, day time.Time, reason string, lotsSold int, pct, pnl float64) Trade {
	holdDays := int(day.Sub(pos.EntryDate).Hours() / 24)
	holdTrading := 0
	for d := pos.EntryDate.AddDate(0, 0, 1); ; d = d.AddDate(0, 0, 1) {
		w := d.Weekday()
		if w != time.Saturday && w != time.Sunday {
			holdTrading++
		}
		if !d.Before(day) {
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
		LotsTotal:       pos.Lot,
		PNLPctNet:       pct,
		PNLRpNet:        pnl,
		HoldDays:        holdDays,
		HoldTradingDays: holdTrading,
	}
}

func removeString(s []string, target string) []string {
	for i, v := range s {
		if v == target {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
