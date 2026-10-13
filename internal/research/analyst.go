package research

import (
	"learn-go/internal/market"
	"learn-go/internal/signals"
)

func GetStockScore(symbol string) (float64, string, float64, float64) {
	bars, err := market.GetBars(symbol, "3mo")
	if err != nil {
		return -1, "", -1, 0
	}

	clean := market.CleanBars(bars)
	if len(clean) < 20 {
		return -1, "Data kurang", -1, 0
	}

	sig := signals.Evaluate(clean, signals.DefaultParams())
	return sig.Score, sig.Verdict, sig.DistToMA, sig.MA20
}

func GetMarketFilterStatus() (bool, string) {
	data, err := market.GetHistoricalPrices("^JKSE")
	if err != nil {
		return true, "⚠️ Gagal mengecek IHSG, asumsikan pasar normal."
	}

	var cleanPrices []float64
	for _, p := range data.Prices {
		if p > 0 {
			cleanPrices = append(cleanPrices, p)
		}
	}

	if len(cleanPrices) < 20 {
		return true, "⚠️ Data IHSG tidak lengkap, asumsikan pasar normal."
	}

	safe, state := signals.MarketSafe(cleanPrices)
	switch state {
	case "DOWNTREND":
		return safe, "🚨 MARKET DOWNTREND / CRASH!"
	case "KOREKSI":
		return safe, "⚠️ MARKET SEDANG KOREKSI WAJAR"
	default:
		return safe, "🟢 MARKET UPTREND (BULLISH)"
	}
}
