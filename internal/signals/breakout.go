package signals

import "learn-go/internal/models"

type breakout20Strategy struct{}

func (s *breakout20Strategy) Name() string   { return "breakout20" }
func (s *breakout20Strategy) MinBars() int   { return 61 }
func (s *breakout20Strategy) Evaluate(bars []models.Bar, ihsgSafe bool) Signal {
	n := len(bars)
	if n < 61 {
		return Signal{Score: 0, Code: "NONE", Low: bars[n-1].Low}
	}

	idxToday := n - 1

	maxHighPrev := MaxHigh(bars, idxToday-20, idxToday-1)
	avgVolPrev := AvgVolume(bars, idxToday-20, idxToday-1)

	today := bars[idxToday]

	if today.Close > maxHighPrev && today.Volume >= 1.5*avgVolPrev && ihsgSafe {
		return Signal{
			Score:   10,
			Code:    "BREAKOUT20",
			Low:     today.Low,
		}
	}
	return Signal{Score: 0, Code: "NONE", Low: today.Low}
}
