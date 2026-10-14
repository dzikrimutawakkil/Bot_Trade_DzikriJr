package signals

import "learn-go/internal/models"

type pullback20Strategy struct{}

func (s *pullback20Strategy) Name() string { return "pullback20" }
func (s *pullback20Strategy) MinBars() int { return 61 }
func (s *pullback20Strategy) Evaluate(bars []models.Bar, ihsgSafe bool) Signal {
	n := len(bars)
	if n < 61 {
		return Signal{Score: 0, Code: "NONE", Low: bars[n-1].Low}
	}

	idxToday := n - 1

	closes := make([]float64, n)
	for i, b := range bars {
		closes[i] = b.Close
	}

	ma20 := SMA(closes, 20, idxToday)
	ma50 := SMA(closes, 50, idxToday)
	ma50Prev := SMA(closes, 50, idxToday-10)

	today := bars[idxToday]

	cond1 := ma20 > ma50
	cond2 := ma50 > ma50Prev
	cond3 := today.Low <= 1.01*ma20
	cond4 := today.Close > ma20
	cond5 := today.Close > today.Open

	if cond1 && cond2 && cond3 && cond4 && cond5 && ihsgSafe {
		return Signal{
			Score: 10,
			Code:  "PULLBACK_MA20",
			Low:   today.Low,
		}
	}
	return Signal{Score: 0, Code: "NONE", Low: today.Low}
}
