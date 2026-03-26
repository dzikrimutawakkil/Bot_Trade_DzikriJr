package research

import (
	"fmt"
    "learn-go/internal/market"
    "learn-go/internal/config"
    "learn-go/internal/utils"
)

func calculateMA(prices []float64, period int) float64 {
	if len(prices) < period { return 0 }
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

func calculateRSI(prices []float64) float64 {
	period := 14
	if len(prices) < period+1 { return 0 }

	var gains, losses float64
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - prices[i-1]
		if diff > 0 { gains += diff } else { losses -= diff }
	}
	
	if losses == 0 { return 100 }
	rs := (gains / float64(period)) / (losses / float64(period))
	return 100 - (100 / (1 + rs))
}

func GetStockScore(symbol string) (float64, string, float64) {
    data, err := market.GetHistoricalPrices(symbol)
    if err != nil { return -1, "", -1 }

    prices := data.Prices
    var cleanPrices []float64
    for _, p := range prices {
        if p > 0 { cleanPrices = append(cleanPrices, p) }
    }
    if len(cleanPrices) < 50 { return -1, "Data kurang", -1 }

    lastPrice := cleanPrices[len(cleanPrices)-1]
    ma20 := calculateMA(cleanPrices, 20)
    rsi := calculateRSI(cleanPrices)

	// Hitung jarak harga ke MA20 dalam persen
    // Contoh: Harga 1050, MA20 1000 -> Jarak 5%
    distToMA := ((lastPrice - ma20) / ma20) * 100

    score := 0.0
    verdict := ""
    
    targetPrice := lastPrice * (1 + config.TPPercent)
    potentialGain := config.TPPercent * 100

    // 1. 🟢 HIJAU (Kondisi Sempurna)
    if lastPrice > ma20 && rsi >= 40 && rsi <= 60 {
        score = 10
        verdict = fmt.Sprintf("🟢 **BELI SEKARANG**\n   🎯 Target: %s (+%.1f%%)\n   🕒 Estimasi: 14-30 hari\n   💡 Alasan: Tren naik dan harga masih wajar.", 
            utils.FormatRupiah(targetPrice), potentialGain)

    // 2. 🟠 SIAGA (Mendekati Hijau)
    // Logika: Harga dikit lagi nembus MA20 (selisih < 2%) DAN RSI sudah mulai kuat (> 38)
    } else if lastPrice < ma20 && lastPrice >= (ma20 * 0.98) && rsi >= 38 {
        score = 8 // Skor tinggi supaya muncul di urutan atas setelah hijau
        verdict = "🟠 **SIAGA SATU**\nAlasan: Dikit lagi harganya nembus area naik. Pantau ketat, siap-siap tarik dana dari RDPU!"

    // 3. 🟡 KUNING (Kepanasan)
    } else if lastPrice > ma20 && rsi > 60 {
        score = 5
        verdict = "🟡 **TUNGGU DULU**\nAlasan: Lagi naik kencang, risiko 'kemahalan' tinggi. Tunggu harga turun dikit (koreksi) baru sikat."

    // 4. 🔴 MERAH (Bahaya/Turun)
    } else {
        score = 1
        verdict = "🔴 **JANGAN BELI**\nAlasan: Trennya masih turun parah. Jangan dilirik dulu sampai ada tanda-tanda harganya mantul."
    }

    return score, verdict, distToMA
}