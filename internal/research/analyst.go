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

    // [UPDATE 1]: Membersihkan Price dan Volume secara bersamaan agar index-nya sinkron
    var cleanPrices []float64
    var cleanVolumes []float64
    for i, p := range data.Prices {
        // Pastikan harga valid dan data volume tersedia di index yang sama
        if p > 0 && i < len(data.Volumes) && data.Volumes[i] > 0 { 
            cleanPrices = append(cleanPrices, p)
            cleanVolumes = append(cleanVolumes, data.Volumes[i])
        }
    }
    
    // Ubah minimal data jadi 20 hari karena kita butuh MA20
    if len(cleanPrices) < 20 { return -1, "Data kurang", -1 }

    // --- KALKULASI INDIKATOR ---
    lastPrice := cleanPrices[len(cleanPrices)-1]
    ma20 := calculateMA(cleanPrices, 20)
    ma5 := calculateMA(cleanPrices, 5) // [UPDATE 2]: Filter trend menengah (Swing)
    
    lastVol := cleanVolumes[len(cleanVolumes)-1]
    avgVol := calculateMA(cleanVolumes, 20) // [UPDATE 3]: Rata-rata volume 20 hari
    
    rsiToday := calculateRSI(cleanPrices)
    rsiYesterday := calculateRSI(cleanPrices[:len(cleanPrices)-1]) // [UPDATE 4]: Momentum naik

    // Hitung jarak harga ke MA20 dalam persen
    distToMA := ((lastPrice - ma20) / ma20) * 100

    score := 0.0
    verdict := ""
    
    targetPrice := lastPrice * (1 + config.TPPercent)
    potentialGain := config.TPPercent * 100

    // --- LOGIKA SKORING ---

    // 1. 🟢 HIJAU (Kondisi Sempurna untuk Swing Trade)
    // Syarat diperketat: Uptrend menengah (ma20 > ma50) & Momentum kuat (rsiToday > rsiYesterday)
    if lastPrice > ma5 && ma5 > ma20 && rsiToday > rsiYesterday && rsiToday >= 40 && rsiToday <= 75 {
        
        // Cek Konfirmasi Volume Bandar
        if lastVol > avgVol {
            score = 10
            verdict = fmt.Sprintf("🟢 **BELI SEKARANG (FAST SWING)**\n   🎯 Target: %s (+%.1f%%)\n   🕒 Estimasi: 2-5 hari\n   💡 Alasan: Momentum jangka pendek kuat (Harga di atas MA5) dan diakumulasi volume.", 
                utils.FormatRupiah(targetPrice), potentialGain)
        } else {
            // Jika teknikal bagus tapi volume sepi, kasih peringatan
            score = 7 
            verdict = "🟡 **NAIK TAPI SEPI**\nAlasan: Secara harga bagus, tapi volume transaksi di bawah rata-rata. Rawan false breakout/bantingan."
        }

    // 2. 🟠 SIAGA (Mendekati Hijau)
    // Logika: Harga dikit lagi nembus MA20 (selisih < 2%) DAN RSI mulai masuk area wajar
    } else if lastPrice < ma20 && lastPrice >= (ma20 * 0.98) && rsiToday >= 38 {
        score = 8 // Skor tinggi supaya muncul di urutan atas setelah hijau
        verdict = "🟠 **SIAGA SATU**\nAlasan: Dikit lagi harganya nembus area naik. Pantau ketat, siap-siap tarik dana dari RDPU!"

    // 3. 🟡 KUNING (Kepanasan / Overbought)
    } else if lastPrice > ma20 && rsiToday > 75 {
        score = 5
        verdict = "🟡 **TUNGGU DULU**\nAlasan: Lagi naik kencang, risiko 'kemahalan' tinggi. Tunggu harga turun dikit (koreksi) baru sikat."

    // 4. 🔴 MERAH (Bahaya/Turun)
    } else {
        score = 1
        verdict = "🔴 **JANGAN BELI**\nAlasan: Trennya masih turun parah (Downtrend). Jangan dilirik dulu sampai ada tanda-tanda harganya mantul."
    }

    return score, verdict, distToMA
}