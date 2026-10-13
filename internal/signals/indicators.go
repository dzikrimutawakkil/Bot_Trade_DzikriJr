package signals

import "learn-go/internal/models"

func SMA(closes []float64, period, idx int) float64 {
	if idx < period-1 {
		return 0
	}
	sum := 0.0
	for i := idx - period + 1; i <= idx; i++ {
		sum += closes[i]
	}
	return sum / float64(period)
}

func MaxHigh(bars []models.Bar, from, to int) float64 {
	max := bars[from].High
	for i := from + 1; i <= to; i++ {
		if bars[i].High > max {
			max = bars[i].High
		}
	}
	return max
}

func AvgVolume(bars []models.Bar, from, to int) float64 {
	n := to - from + 1
	if n <= 0 {
		return 0
	}
	sum := 0.0
	for i := from; i <= to; i++ {
		sum += bars[i].Volume
	}
	return sum / float64(n)
}
