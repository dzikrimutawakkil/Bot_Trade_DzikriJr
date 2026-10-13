package pricing

import "math"

func RoundToFraction(price float64) float64 {
	var fraction float64

	switch {
	case price < 50:
		fraction = 1
	case price < 200:
		fraction = 1
	case price < 500:
		fraction = 2
	case price < 2000:
		fraction = 5
	case price < 5000:
		fraction = 10
	default:
		fraction = 25
	}

	return math.Round(price/fraction) * fraction
}

func CalculateNetPNL(entryPrice float64, currentPrice float64, buyFee float64, sellFee float64) float64 {
	totalBuyCapital := entryPrice * (1 + buyFee)
	netSellValue := currentPrice * (1 - sellFee)
	return ((netSellValue - totalBuyCapital) / totalBuyCapital) * 100
}
