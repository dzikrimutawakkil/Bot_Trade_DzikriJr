package signals

import (
	"sort"
	"time"

	"learn-go/internal/models"
)

// WeeklyCandidate is one stock that passed the uptrend filter, ready for ranking.
type WeeklyCandidate struct {
	Symbol      string
	Open        float64
	Close       float64
	MA5         float64
	MA20        float64
	MA50        float64
	MomentumPct float64
}

// SelectWeeklyCandidates scans the universe on `day`, keeps stocks whose
// MA5 > MA20 > MA50, ranks by (close - MA5) / MA5 descending, returns the
// top N. Pure function: no I/O, no state, deterministic.
func SelectWeeklyCandidates(allBars map[string][]models.Bar, symbols []string, day time.Time, topN int) []WeeklyCandidate {
	dayStr := day.Format("2006-01-02")
	candidates := make([]WeeklyCandidate, 0, len(symbols))

	for _, sym := range symbols {
		bars, ok := allBars[sym]
		if !ok || len(bars) == 0 {
			continue
		}

		idx := -1
		for i, b := range bars {
			if b.Time.Format("2006-01-02") == dayStr {
				idx = i
				break
			}
		}
		if idx < 50 {
			continue
		}

		closes := make([]float64, idx+1)
		for i := 0; i <= idx; i++ {
			closes[i] = bars[i].Close
		}

		ma5 := SMA(closes, 5, idx)
		ma20 := SMA(closes, 20, idx)
		ma50 := SMA(closes, 50, idx)
		if ma5 == 0 || ma20 == 0 || ma50 == 0 {
			continue
		}
		if !(ma5 > ma20 && ma20 > ma50) {
			continue
		}

		// Intraday volatility filter: avg((High-Low)/Close) last 10 days
		if idx >= 10 {
			sumVol := 0.0
			for i := idx - 9; i <= idx; i++ {
				if bars[i].Close > 0 {
					sumVol += (bars[i].High - bars[i].Low) / bars[i].Close
				}
			}
			avgVol := sumVol / 10.0
			if avgVol > 0.06 { // Test: 6% threshold
				continue
			}
		}

		close := closes[idx]
		momentum := (close - ma5) / ma5 * 100
		candidates = append(candidates, WeeklyCandidate{
			Symbol:      sym,
			Open:        bars[idx].Open,
			Close:       close,
			MA5:         ma5,
			MA20:        ma20,
			MA50:        ma50,
			MomentumPct: momentum,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].MomentumPct > candidates[j].MomentumPct
	})

	if topN > 0 && len(candidates) > topN {
		candidates = candidates[:topN]
	}
	return candidates
}

// TrendBroken returns true when MA5 < MA20 at the given day for `sym`.
// Used to exit positions on weekly rebalance when uptrend collapses.
func TrendBroken(bars []models.Bar, day time.Time) bool {
	dayStr := day.Format("2006-01-02")
	idx := -1
	for i, b := range bars {
		if b.Time.Format("2006-01-02") == dayStr {
			idx = i
			break
		}
	}
	if idx < 20 {
		return false
	}
	closes := make([]float64, idx+1)
	for i := 0; i <= idx; i++ {
		closes[i] = bars[i].Close
	}
	ma5 := SMA(closes, 5, idx)
	ma20 := SMA(closes, 20, idx)
	return ma5 > 0 && ma20 > 0 && ma5 < ma20
}
