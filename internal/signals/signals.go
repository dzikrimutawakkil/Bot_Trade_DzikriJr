package signals

import (
	"learn-go/internal/models"
	"learn-go/internal/pricing"
	"math"
)

type Params struct {
	MinScore      float64
	PumpFilterPct float64
	ARBDropPct    float64
}

func DefaultParams() Params {
	return Params{
		MinScore:      8.0,
		PumpFilterPct: 3.5,
		ARBDropPct:    10.0,
	}
}

type Signal struct {
	Score    float64
	Code     string
	Verdict  string
	DistToMA float64
	MA20     float64
	Low      float64
}

func calcMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

func calcAvgVol(vols []float64, period int) float64 {
	if len(vols) < period {
		return 0
	}
	sum := 0.0
	for i := len(vols) - period; i < len(vols); i++ {
		sum += vols[i]
	}
	return sum / float64(period)
}

func Evaluate(bars []models.Bar, p Params) Signal {
	if len(bars) < 20 {
		return Signal{Score: -1, Code: "NO_DATA", Verdict: "Data kurang"}
	}

	closes := make([]float64, len(bars))
	vols := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
		vols[i] = b.Volume
	}

	ma20 := calcMA(closes, 20)
	ma5 := calcMA(closes, 5)
	avgVol := calcAvgVol(vols, 20)

	idxToday := len(bars) - 1
	idxYest := len(bars) - 2

	today := bars[idxToday]
	yest := bars[idxYest]

	todayClose := today.Close
	todayOpen := today.Open
	todayHigh := today.High
	todayLow := today.Low
	todayVol := today.Volume

	yestClose := yest.Close

	distToMA := ((todayClose - ma20) / ma20) * 100

	prevClose := closes[len(closes)-3]
	yestChange := ((yestClose - prevClose) / prevClose) * 100
	if p.ARBDropPct > 0 && yestChange <= -p.ARBDropPct {
		return Signal{
			Score: -1, Code: "BLOCKED_ARB",
			Verdict:  "⛔ ANTI-ARB — saham turun lebih dari 10% kemarin",
			DistToMA: distToMA, MA20: ma20, Low: todayLow,
		}
	}

	pumpFromLow := ((todayClose - todayLow) / todayLow) * 100
	if p.PumpFilterPct > 0 && pumpFromLow > p.PumpFilterPct {
		return Signal{
			Score: -1, Code: "BLOCKED_PUMP",
			Verdict:  "⛔ ANTI-DISTRIBUTION TRAP — pump > 3.5% dari low",
			DistToMA: distToMA, MA20: ma20, Low: todayLow,
		}
	}

	body := math.Abs(todayClose - todayOpen)
	lowerWick := math.Min(todayOpen, todayClose) - todayLow
	dailyRange := todayHigh - todayLow
	if dailyRange == 0 {
		dailyRange = 1
	}

	isHammer := todayLow <= ma20 &&
		todayClose > ma20 &&
		lowerWick >= (2*body) &&
		(todayHigh-todayClose) <= (dailyRange*0.3)

	yestBodyMidpoint := (yest.Open + yestClose) / 2.0
	isGreenBounce := todayClose > todayOpen &&
		todayClose > yestBodyMidpoint &&
		todayVol > yest.Volume &&
		distToMA <= 4.0 &&
		todayLow >= ma20*0.98

	if isHammer {
		return Signal{
			Score: 10, Code: "HAMMER",
			Verdict:  "🟢 CONFIRMED BoW (HIGH-QUALITY HAMMER)",
			DistToMA: distToMA, MA20: ma20, Low: todayLow,
		}
	}
	if isGreenBounce {
		return Signal{
			Score: 10, Code: "GREEN_BOUNCE",
			Verdict:  "🟢 CONFIRMED BoW (STRONG GREEN BOUNCE)",
			DistToMA: distToMA, MA20: ma20, Low: todayLow,
		}
	}
	if todayClose >= ma20 && todayClose < ma5 && distToMA <= 3.0 && todayVol < avgVol {
		return Signal{
			Score: 7, Code: "SIAGA",
			Verdict:  "🟡 SIAGA PANTULAN (BELUM KONFIRMASI)",
			DistToMA: distToMA, MA20: ma20, Low: todayLow,
		}
	}
	if distToMA > 5.0 {
		return Signal{
			Score: 4, Code: "PUCUK",
			Verdict:  "🔴 RAWAN PUCUK / FOMO",
			DistToMA: distToMA, MA20: ma20, Low: todayLow,
		}
	}
	if todayClose < ma20 {
		return Signal{
			Score: 1, Code: "PISAU",
			Verdict:  "💀 PISAU JATUH (DOWNTREND)",
			DistToMA: distToMA, MA20: ma20, Low: todayLow,
		}
	}
	return Signal{
		Score: 5, Code: "TANGGUNG",
		Verdict:  "⚪ TANGGUNG / SIDEWAYS",
		DistToMA: distToMA, MA20: ma20, Low: todayLow,
	}
}

func MarketSafe(ihsgCloses []float64) (bool, string) {
	n := len(ihsgCloses)
	if n < 20 {
		return true, "UPTREND"
	}
	current := ihsgCloses[n-1]
	ma20 := calcMA(ihsgCloses, 20)
	ma5 := calcMA(ihsgCloses, 5)

	if current < ma20 {
		return false, "DOWNTREND"
	}
	if current <= ma5 {
		return true, "KOREKSI"
	}
	return true, "UPTREND"
}

func StructuralCutLoss(signalLow float64) float64 {
	return pricing.RoundToFraction(signalLow * 0.99)
}

func TP1Price(entry float64) float64 {
	return pricing.RoundToFraction(entry * 1.04)
}

func NetPNL(entry, price float64) float64 {
	return pricing.CalculateNetPNL(entry, price, 0.0015, 0.0025)
}
